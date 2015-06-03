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
package controller

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"fmt"
	ca "github.com/NEU-SNS/ReverseTraceroute/lib/cache"
	capi "github.com/NEU-SNS/ReverseTraceroute/lib/controllerapi"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"os"
	"sync"
	"time"
)

const (
	ID = "ID"
)

func getUUID() string {
	return uuid.NewUUID().String()
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
	rpc      dm.DialFunc
}

var controller controllerT

func (c *controllerT) getRequests() int64 {
	c.mu.Lock()
	req := c.requests
	c.mu.Unlock()
	return req
}

func (c *controllerT) addRequest() {
	c.mu.Lock()
	c.requests += 1
	c.mu.Unlock()
}

func (c *controllerT) addTime(t time.Duration) {
	c.mu.Lock()
	c.time += t
	c.mu.Unlock()
}

func (c *controllerT) getTime() time.Duration {
	c.mu.Lock()
	time := c.time
	c.mu.Unlock()
	return time
}

func (c *controllerT) addReqStats(req Request) {
	c.mu.Lock()
	c.time += req.Dur
	c.requests += 1
	c.mu.Unlock()
}

func (c *controllerT) getStatsInfo() (t time.Duration, req int64) {
	c.mu.Lock()
	t, req = c.time, c.requests
	c.mu.Unlock()
	return
}

func HandleSig(sig os.Signal) {
	controller.handleSig(sig)
}

func (c *controllerT) handleSig(sig os.Signal) {
	c.db.Close()
}

func (c *controllerT) getStats() dm.Stats {
	utime := time.Since(c.startTime)
	t, req := c.getStatsInfo()
	var tt time.Duration
	if t == 0 {
		tt = 0
	} else {
		avg := int64(t) / int64(req)
		tt = time.Duration(avg)
	}
	s := dm.Stats{StartTime: c.startTime.UnixNano(),
		UpTime: utime.Nanoseconds(), Requests: req,
		TotReqTime: t.Nanoseconds(), AvgReqTime: tt.Nanoseconds()}
	return s
}

func (c *controllerT) startRpc(eChan chan error) {
	var addr string
	if c.config.Local.AutoConnect {
		saddr, err := util.GetBindAddr()
		if err != nil {
			eChan <- err
			return
		}
		addr = fmt.Sprintf("%s:%d", saddr, 35000)
	} else {
		addr = c.config.Local.Addr
	}
	glog.Infof("Conecting to: %s", addr)
	l, e := net.Listen(c.config.Local.Proto, addr)
	if e != nil {
		glog.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	glog.Infof("Controller started, listening on: %s", addr)
	err := c.server.Serve(l)
	if err != nil {
		eChan <- err
	}
}

func (c *controllerT) doStats(ctx con.Context, sa *dm.StatsArg) (sr *dm.StatsReturn, err error) {
	st := time.Now()
	uuid := getUUID()
	glog.Infof("%s: Ping starting", uuid)
	nctx := con.WithValue(ctx, ID, uuid)
	sr = new(dm.StatsReturn)
	s, mt, err := c.getService(sa.Service)
	if err != nil {
		sr.Ret = makeErrorReturn(st)
		return
	}
	ip, err := s.GetIp()
	if err != nil {
		return nil, err
	}
	err = mt.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
	if err != nil {
		sr.Ret = makeErrorReturn(st)
		return
	}
	sr.Stats, err = mt.Stats(nctx, sa)
	if err != nil {
		sr.Ret = makeErrorReturn(st)
		return
	}
	sr.Ret = makeSuccessReturn(st)
	return
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
	err = mt.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
	if err != nil {
		return nil, err
	}
	return mt, nil
}

func (c *controllerT) doPing(ctx con.Context, pa *dm.PingArg) (pr *dm.PingReturn, err error) {
	st := time.Now()
	uuid := getUUID()
	glog.Infof("%s: Ping starting", uuid)
	nctx := con.WithValue(ctx, ID, uuid)
	pr = new(dm.PingReturn)

	mt, err := c.getMeasurementTool(pa.Service)
	if err != nil {
		pr.Ret = makeErrorReturn(st)
		return
	}
	pr.Ping, err = mt.Ping(nctx, pa)
	if err != nil {
		pr.Ret = makeErrorReturn(st)
		return
	}
	pr.Ret = makeSuccessReturn(st)
	return
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
	lhops := make([]int64, len(hops))
	for i, hop := range hops {
		ip, err := util.IpStringToInt64(hop.Addr)
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
	uuid := getUUID()
	glog.Infof("%s: Traceroute starting", uuid)
	nctx := con.WithValue(ctx, ID, uuid)
	tr = new(dm.TracerouteReturn)
	makeRequest := !ta.CheckCache
	if ta.CheckCache {
		trace, e := c.db.GetTRBySrcDstWithStaleness(ta.Host, ta.Dst, da.Staleness(ta.Staleness))
		if e == nil {
			glog.Infof("Got traceroute from db: %v", trace)
			tr.Traceroute = trace
			tr.Ret = makeSuccessReturn(st)
			return
		}
		glog.Errorf("Failed to get traceroute from db: %v, got: %v", e, trace)
		makeRequest = true
	}

	if makeRequest {
		mt, e := c.getMeasurementTool(ta.Service)
		if e != nil {
			tr.Ret = makeErrorReturn(st)
			return
		}
		trace, e := mt.Traceroute(nctx, ta)
		if e != nil {
			tr.Ret = makeErrorReturn(st)
			return
		}
		tr.Traceroute = makeMTraceroute(trace, ta.Service)
		if tr.Traceroute == nil {
			tr.Ret = makeErrorReturn(st)
			err = fmt.Errorf("Invalid traceroute received")
			return
		}
		go func() {
			e = c.db.StoreTraceroute(trace, ta.Service)
			if e != nil {
				glog.Errorf("Failed to store traceroute: %v", e)
			}
		}()
		tr.Ret = makeSuccessReturn(st)
		return
	}

	tr.Ret = makeErrorReturn(st)
	err = fmt.Errorf("doTraceroute failed to find in cache or remote")
	return
}

// VPs only apply to planet-lab currently, so the get VP type calls
// will be hardcoded to look for planet-lab vps
func (c *controllerT) getVP(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetVP(ctx, arg)

	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
}

func (c *controllerT) getAllVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetAllVPs(ctx, arg)
	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
}

func (c *controllerT) getSpoofingVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetSpoofingVPs(ctx, arg)
	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
}

func (c *controllerT) getTimeStampVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetTimeStampVPs(ctx, arg)
	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
}

func (c *controllerT) getRecordRouteVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetRecordRouteVPs(ctx, arg)
	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
}

func (c *controllerT) getActiveVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	plc, s, err := c.router.GetClient(dm.ServiceT_PLANET_LAB)
	if err != nil {
		return nil, err
	}
	if plcl, ok := plc.(plClient); ok {
		ip, err := s.GetIp()
		if err != nil {
			return nil, err
		}
		err = plcl.Connect(ip, time.Duration(c.config.Local.ConnTimeout)*time.Second)
		if err != nil {
			return nil, err
		}
		return plcl.GetActiveVPs(ctx, arg)
	}
	return nil, fmt.Errorf("Error tried to get vp from incorrect service")
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

func Start(c Config, db da.DataProvider, cache ca.Cache) chan error {
	errChan := make(chan error, 2)
	if db == nil {
		glog.Errorf("Nil db in Controller Start")
		errChan <- errors.New("Controller Start, nil DB")
		return errChan
	}
	if cache == nil {
		glog.Errorf("Nil cache in Controller start")
		errChan <- errors.New("Controller Start, nil Cache")
		return errChan
	}
	controller.config = c
	controller.startTime = time.Now()
	controller.db = db
	controller.cache = cache
	controller.router = createRouter()
	controller.router.RegisterServices(c.Local.Services...)
	var opts []grpc.ServerOption
	controller.server = grpc.NewServer(opts...)
	capi.RegisterControllerServer(controller.server, &controller)
	go controller.startRpc(errChan)
	return errChan
}
