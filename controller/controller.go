/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package controller is the library for creating a central controller
package controller

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	ca "github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/controllerapi"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/router"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, getName())
	rpcCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: getName(),
		Subsystem: "rpc",
		Name:      "count",
		Help:      "Count of Rpc Calls sent",
	})
	timeoutCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: getName(),
		Subsystem: "rpc",
		Name:      "timeout_count",
		Help:      "Count of Rpc Timeouts",
	})
	errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: getName(),
		Subsystem: "rpc",
		Name:      "error_count",
		Help:      "Count of Rpc Errors",
	})
)
var id = rand.Uint32()

func getName() string {
	name, err := os.Hostname()
	if err != nil {
		return fmt.Sprintf("controller_%d", id)
	}
	return fmt.Sprintf("controller_%s", strings.Replace(name, ".", "_", -1))
}

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(rpcCounter)
	prometheus.MustRegister(timeoutCounter)
	prometheus.MustRegister(errorCounter)
}

type controllerT struct {
	config    Config
	db        *da.DataAccess
	cache     ca.Cache
	router    router.Router
	startTime time.Time
	server    *grpc.Server
	mu        sync.Mutex
	//the mutex protects the following
	requests int64
	time     time.Duration
}

var controller controllerT

// HandleSig handles and signals received from the OS
func HandleSig(sig os.Signal) {
	controller.handleSig(sig)
}

func (c *controllerT) handleSig(sig os.Signal) {
	log.Infof("Got signal: %v", sig)
	c.stop()
}

func (c *controllerT) startRPC(eChan chan error) {
	addr := fmt.Sprintf("%s:%d", *c.config.Local.Addr,
		*c.config.Local.Port)
	log.Infof("Conecting to: %s", addr)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	log.Infof("Controller started, listening on: %s", addr)
	err := c.server.Serve(l)
	if err != nil {
		eChan <- err
	}
}

type pingFunc func(con.Context, <-chan []*dm.PingMeasurement) <-chan *dm.Ping
type pingStep func(pingFunc) pingFunc
type traceFunc func(con.Context, <-chan []*dm.TracerouteMeasurement) <-chan *dm.Traceroute
type traceStep func(traceFunc) traceFunc

type routed struct {
	r router.Router
}

func errorAllPing(err error, out chan<- *dm.Ping, ps []*dm.PingMeasurement) {
	for _, p := range ps {
		out <- &dm.Ping{
			Src:   p.Src,
			Dst:   p.Dst,
			Error: err.Error(),
		}
	}
}

func errorAllTrace(err error, out chan<- *dm.Traceroute, ts []*dm.TracerouteMeasurement) {
	for _, t := range ts {
		out <- &dm.Traceroute{
			Src:   t.Src,
			Dst:   t.Dst,
			Error: err.Error(),
		}
	}
}

func (r routed) pingMeas(ctx con.Context, pm <-chan []*dm.PingMeasurement) <-chan *dm.Ping {
	ret := make(chan *dm.Ping)
	go func() {
		var wg sync.WaitGroup
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case pms, ok := <-pm:
				if !ok {
					// Our inbound channel is closed so break this loop
					break loop
				}
				mts := make(map[router.ServiceDef][]*dm.PingMeasurement)
				for _, p := range pms {
					srcs, _ := util.Int32ToIPString(p.Src)
					serv, err := r.r.GetService(srcs)
					if err != nil {
						ret <- &dm.Ping{
							Src:   p.Src,
							Dst:   p.Dst,
							Error: err.Error(),
						}
						continue
					}
					mts[serv] = append(mts[serv], p)
				}
				for serv, meas := range mts {
					mt, err := r.r.GetMT(serv)
					if err != nil {
						log.Error(err)
						errorAllPing(err, ret, meas)
						continue
					}
					wg.Add(1)
					go func(t router.MeasurementTool, m []*dm.PingMeasurement) {
						defer wg.Done()
						ps, err := t.Ping(ctx, &dm.PingArg{
							Pings: m})
						if err != nil {
							log.Error(err)
							errorAllPing(err, ret, m)
							return
						}
						for ping := range ps {
							ret <- ping
						}
					}(mt, meas)
				}
			}
		}
		// This waits when the inbound channel is closed
		// The wg is increased every time a separate ping request is issued
		wg.Wait()
		close(ret)
	}()
	return ret
}

func (c *controllerT) doPing(ctx con.Context, pm []*dm.PingMeasurement) <-chan *dm.Ping {
	do := pingCache{c: c.cache}.pingCacheStep(
		pingDB{db: c.db}.pingDBStep(routed{r: c.router}.pingMeas))
	next := make(chan []*dm.PingMeasurement)
	res := do(ctx, next)
	go func() {
		next <- pm
		close(next)
	}()
	return res
}

func (r routed) traceMeas(ctx con.Context, tm <-chan []*dm.TracerouteMeasurement) <-chan *dm.Traceroute {
	ret := make(chan *dm.Traceroute)
	go func() {
		var wg sync.WaitGroup
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case tms, ok := <-tm:
				if !ok {
					break loop
				}
				sds := make(map[router.ServiceDef][]*dm.TracerouteMeasurement)
				for _, t := range tms {
					srcs, _ := util.Int32ToIPString(t.Src)
					serv, err := r.r.GetService(srcs)
					if err != nil {
						ret <- &dm.Traceroute{
							Src:   t.Src,
							Dst:   t.Dst,
							Error: err.Error(),
						}
						continue
					}
					sds[serv] = append(sds[serv], t)
				}
				for serv, meas := range sds {
					mt, err := r.r.GetMT(serv)
					if err != nil {
						log.Error(err)
						errorAllTrace(err, ret, meas)
						continue
					}
					wg.Add(1)
					go func(t router.MeasurementTool, tt []*dm.TracerouteMeasurement) {
						defer wg.Done()
						ts, err := t.Traceroute(ctx, &dm.TracerouteArg{
							Traceroutes: tt,
						})
						if err != nil {
							log.Error(err)
							errorAllTrace(err, ret, tt)
							return
						}
						for trace := range ts {
							ret <- trace
						}
					}(mt, meas)
				}
			}
		}
		wg.Wait()
		close(ret)
	}()
	return ret
}

func (c *controllerT) doTraceroute(ctx con.Context, tm []*dm.TracerouteMeasurement) <-chan *dm.Traceroute {
	log.Infof("%s: Traceroute starting")
	do := traceCache{c: c.cache}.traceCacheStep(
		traceDB{db: c.db}.traceDBStep(routed{r: c.router}.traceMeas))
	next := make(chan []*dm.TracerouteMeasurement)
	res := do(ctx, next)
	go func() {
		next <- tm
		close(next)
	}()
	return res
}

func (c *controllerT) fetchVPs(ctx con.Context, gvp *dm.VPRequest) (*dm.VPReturn, error) {
	mts := c.router.All()
	var ret dm.VPReturn
	for _, mt := range mts {
		vpc, err := mt.GetVPs(ctx, gvp)
		if err != nil {
			return nil, err
		}
		for vp := range vpc {
			ret.Vps = append(ret.Vps, vp.Vps...)
		}
		mt.Close()
	}
	go func() {
		data, err := proto.Marshal(&ret)
		if err != nil {
			log.Error(err)
			return
		}
		// Cache for 5 min
		c.cache.SetWithExpire("all_vps", data, 5*60)
	}()
	return &ret, nil
}

func (c *controllerT) doGetVPs(ctx con.Context, gvp *dm.VPRequest) (*dm.VPReturn, error) {
	res, err := c.cache.Get("all_vps")
	if err != nil && err != ca.ErrorCacheMiss {
		log.Error(err)
		return nil, err
	}
	var ret dm.VPReturn
	if err == ca.ErrorCacheMiss {
		return c.fetchVPs(ctx, gvp)
	}
	err = proto.Unmarshal(res.Value(), &ret)
	if err != nil {
		return nil, err
	}
	if len(ret.Vps) == 0 {
		return c.fetchVPs(ctx, gvp)
	}
	return &ret, nil
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

func (c *controllerT) stop() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *controllerT) run(ec chan error, con Config, db *da.DataAccess, cache ca.Cache, r router.Router) {
	controller.config = con
	controller.db = db
	controller.cache = cache
	controller.router = r
	if db == nil {
		log.Errorf("Nil db in Controller Start")
		c.stop()
		ec <- errors.New("Controller Start, nil DB")
		return
	}
	if cache == nil {
		log.Errorf("Nil cache in Controller start")
		c.stop()
		ec <- errors.New("Controller Start, nil Cache")
		return
	}
	if r == nil {
		log.Errorf("Nil router in Controller start")
		c.stop()
		ec <- errors.New("Controller Start, nil router")
		return
	}
	controller.server = grpc.NewServer()
	controllerapi.RegisterControllerServer(controller.server, c)
	go controller.startRPC(ec)
}

// Start starts a central controller with the given configuration
func Start(c Config, db *da.DataAccess, cache ca.Cache, r router.Router) chan error {
	log.Info("Starting controller")
	http.Handle("/metrics", prometheus.Handler())
	go startHTTP(*c.Local.PProfAddr)
	errChan := make(chan error, 2)
	go controller.run(errChan, c, db, cache, r)
	return errChan
}
