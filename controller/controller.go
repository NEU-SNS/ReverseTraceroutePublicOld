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
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/prometheus/client_golang/prometheus"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"

	_ "net/http/pprof"
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
var id uint32 = rand.Uint32()

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
	db        da.DataProvider
	cache     ca.Cache
	router    Router
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
	var addr string
	if *c.config.Local.AutoConnect {
		saddr, err := util.GetBindAddr()
		if err != nil {
			eChan <- err
			return
		}
		addr = fmt.Sprintf("%s:%d", saddr, 35000)
	} else {
		addr = *c.config.Local.Addr
	}
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

func (c *controllerT) getMeasurementTool(serv dm.ServiceT) (MeasurementTool, error) {
	s, mt, err := c.getService(serv)
	if err != nil {
		return nil, err
	}
	ip, err := s.GetIp()
	if err != nil {
		return nil, err
	}
	err = mt.Connect(ip, time.Duration(*c.config.Local.ConnTimeout)*time.Second)
	if err != nil {
		return nil, err
	}
	return mt, nil
}

type pingFunc func(con.Context, <-chan []*dm.PingMeasurement) <-chan *dm.Ping
type pingStep func(pingFunc) pingFunc

func pingMeas(ctx con.Context, pm <-chan []*dm.PingMeasurement) <-chan *dm.Ping {
	ret := make(chan *dm.Ping)
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ret)
				return
			case <-pm:
				/*
					Do stuff to run the measurements
					Return the results back
				*/
				close(ret)
				return
			}
		}
	}()
	return ret
}

func (c *controllerT) doPing(ctx con.Context, pm []*dm.PingMeasurement) <-chan *dm.Ping {
	log.Infof("%s: Ping starting")
	do := pingCache{c: c.cache}.pingCacheStep(
		pingDB{db: c.db}.pingDBStep(pingMeas))
	next := make(chan []*dm.PingMeasurement)
	res := do(ctx, next)
	go func() {
		next <- pm
		close(next)
	}()
	return res
}

func makeMTraceroute(t *dm.Traceroute, s dm.ServiceT) *dm.MTraceroute {
	mt := new(dm.MTraceroute)
	mt.Service = s
	mt.Date = time.Unix(t.Start.Sec, util.MicroToNanoSec(t.Start.Usec)).Unix()
	mt.Src = t.Src
	mt.Dst = t.Dst
	hops := t.GetHops()
	if hops == nil {
		return nil
	}
	lhops := make([]uint32, len(hops))
	for i, hop := range hops {
		ip, err := util.IPStringToInt32(hop.Addr)
		if err != nil {
			return nil
		}
		lhops[i] = ip
	}
	mt.Hops = lhops
	return mt
}

func (c *controllerT) doTraceroute(ctx con.Context, ta *dm.TracerouteArg) (tr *dm.TracerouteReturn, err error) {
	st := time.Now()
	log.Infof("%s: Traceroute starting")
	tr = new(dm.TracerouteReturn)
	tr.Ret = makeErrorReturn(st)
	err = fmt.Errorf("doTraceroute failed to find in cache or remote")
	return
}

func (c *controllerT) doGetVPs(ctx con.Context, gvp *dm.VPRequest) (vpr *dm.VPReturn, err error) {
	return new(dm.VPReturn), nil
}

func makeSuccessReturn(t time.Time) *dm.ReturnT {
	mr := new(dm.ReturnT)
	mr.Dur = time.Since(t).Nanoseconds()
	mr.Status = dm.MRequestStatus_SUCCESS
	return mr
}

func makeErrorReturn(t time.Time) *dm.ReturnT {
	mr := new(dm.ReturnT)
	mr.Dur = time.Since(t).Nanoseconds()
	mr.Status = dm.MRequestStatus_ERROR
	return mr
}

func (c *controllerT) getService(s dm.ServiceT) (*dm.Service, MeasurementTool, error) {
	return c.router.GetService(s)
}

func startHttp(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

func (c *controllerT) stop() {
	if c.db != nil {
		c.db.Close()
	}
}

func (c *controllerT) run(ec chan error, con Config, db da.DataProvider, cache ca.Cache) {
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
	controller.config = con
	controller.startTime = time.Now()
	controller.db = db
	controller.cache = cache
	controller.router = createRouter()
	var opts []grpc.ServerOption
	controller.server = grpc.NewServer(opts...)
	go controller.startRPC(ec)
}

// Start starts a central controller with the given configuration
func Start(c Config, db da.DataProvider, cache ca.Cache) chan error {
	log.Info("Starting controller")
	http.Handle("/metrics", prometheus.Handler())
	go startHttp(*c.Local.PProfAddr)
	errChan := make(chan error, 2)
	go controller.run(errChan, c, db, cache)
	return errChan
}
