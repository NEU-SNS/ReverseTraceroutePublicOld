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
	capi "github.com/NEU-SNS/ReverseTraceroute/lib/controllerapi"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
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
	db        da.DataAccess
	router    Router
	startTime time.Time
	server    *grpc.Server
	mu        sync.Mutex
	//the mutex protects the following
	requests int64
	time     time.Duration
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
	l, e := net.Listen(c.config.Local.Proto, c.config.Local.Addr)
	if e != nil {
		glog.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	glog.Infof("Controller started, listening on: %s", c.config.Local.Addr)
	err := c.server.Serve(l)
	eChan <- err
}

func (c *controllerT) doStats(ctx con.Context, sa *dm.StatsArg) (sr *dm.StatsReturn, err error) {
	st := time.Now()
	uuid := getUUID()
	glog.Infof("%s: Ping starting")
	nctx := con.WithValue(ctx, ID, uuid)
	sr = new(dm.StatsReturn)
	s, mt, err := c.getService(sa.Service)
	if err != nil {
		sr.Ret = makeErrorReturn(st)
		return
	}
	err = mt.Connect(s.GetIp())
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

func (c *controllerT) doPing(ctx con.Context, pa *dm.PingArg) (pr *dm.PingReturn, err error) {
	st := time.Now()
	uuid := getUUID()
	glog.Infof("%s: Ping starting", uuid)
	nctx := con.WithValue(ctx, ID, uuid)
	pr = new(dm.PingReturn)
	s, mt, err := c.getService(pa.Service)
	if err != nil {
		pr.Ret = makeErrorReturn(st)
		return
	}
	err = mt.Connect(s.GetIp())
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

func (c *controllerT) doTraceroute(ctx con.Context, ta *dm.TracerouteArg) (tr *dm.TracerouteReturn, err error) {
	st := time.Now()
	uuid := getUUID()
	glog.Infof("%s: Ping starting")
	nctx := con.WithValue(ctx, ID, uuid)
	tr = new(dm.TracerouteReturn)
	s, mt, err := c.getService(ta.Service)
	if err != nil {
		tr.Ret = makeErrorReturn(st)
		return
	}
	err = mt.Connect(s.GetIp())
	if err != nil {
		tr.Ret = makeErrorReturn(st)
		return
	}

	tr.Traceroute, err = mt.Traceroute(nctx, ta)
	if err != nil {
		tr.Ret = makeErrorReturn(st)
		return
	}
	tr.Ret = makeSuccessReturn(st)
	return
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

func Start(c Config, db da.DataAccess) chan error {
	errChan := make(chan error, 1)
	if db == nil {
		glog.Errorf("Nil db in Controller Start")
		errChan <- errors.New("Controller Start, nil DB")
		return errChan
	}
	controller.config = c
	controller.startTime = time.Now()
	controller.db = db
	controller.router = createRouter()
	controller.router.RegisterServices(
		db.GetServices("")...)

	var opts []grpc.ServerOption
	controller.server = grpc.NewServer(opts...)
	capi.RegisterControllerServer(controller.server, &controller)
	go controller.startRpc(errChan)
	return errChan
}
