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
package plcontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/lib/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/go-fsnotify/fsnotify"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"net"
	"os"
	"sync"
	"time"
)

type plControllerT struct {
	startTime time.Time
	spid      int
	server    *grpc.Server
	config    Config
	sc        scamper.Config
	mp        mproc.MProc
	db        da.VantagePointProvider
	w         *fsnotify.Watcher
	conf      Config
	reqCount  *ReqCount
	mu        sync.Mutex
	//the mutex protects the following
	requests int64
	time     time.Duration

	rw sync.RWMutex
	//rwmutex protext the socks
	socks map[string]scamper.Socket
}

type ReqCount struct {
	val uint32
	mu  sync.Mutex
}

func (r *ReqCount) inc() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.val += 1
}

func (r *ReqCount) dec() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.val -= 1
}

func handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	switch err.(type) {
	default:
		return false
	case *os.PathError:
		return true
	}

}

var plController plControllerT

func (c *plControllerT) getStatsInfo() (t time.Duration, req int64) {
	c.mu.Lock()
	t, req = c.time, c.requests
	c.mu.Unlock()
	return
}

func (c *plControllerT) getStats() dm.Stats {
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

func (c *plControllerT) runPing(pa dm.PingArg) (dm.Ping, error) {
	glog.Infof("Running ping for: %v", pa)
	timeout := pa.Timeout
	if timeout == 0 {
		timeout = c.conf.Local.Timeout
	}
	sTime := time.Now()
	ret := dm.Ping{}
	soc, err := c.getSocket(pa.Host)
	if err != nil {
		return ret, err
	}
	com, err := scamper.NewCmd(pa)
	if err != nil {
		return ret, err
	}
	cl := scamper.NewClient(soc, com)
	ec := make(chan error, 1)
	dc := make(chan struct{}, 1)
	go cl.IssueCmd(ec, dc)
	select {
	case err := <-ec:
		return ret, err
	case <-time.After(time.Second * time.Duration(c.conf.Local.Timeout)):
		cl.CancelCmd()
		return ret, fmt.Errorf("Ping timed out")
	case <-dc:
		break
	}
	resps := cl.GetResponses()
	var dw util.UUDecodingWriter
	for _, r := range resps {
		glog.Infof("Decoding and writing: %s", r.Bytes())
		_, err := r.WriteTo(&dw)
		if err != nil {
			return ret, err
		}
	}
	res, err := c.convertWarts(dw.Bytes())
	if err != nil {
		glog.Errorf("Error converting warts: %v", err)
		return ret, err
	}
	err = decodeResponse(&res, &ret)
	if err != nil {
		glog.Errorf("Failed to decode ping with err: %v", err)
		return ret, err
	}
	c.increaseStats(sTime)
	glog.Infof("Ping done: %v", ret)
	return ret, nil
}

func (c *plControllerT) runTraceroute(ta dm.TracerouteArg) (dm.Traceroute, error) {
	glog.Infof("Running traceroute for: %v", ta)
	timeout := ta.Timeout
	if timeout == 0 {
		timeout = c.conf.Local.Timeout
	}
	sTime := time.Now()
	ret := dm.Traceroute{}
	soc, err := c.getSocket(ta.Host)
	if err != nil {
		return ret, err
	}
	com, err := scamper.NewCmd(ta)
	cl := scamper.NewClient(soc, com)
	ec := make(chan error, 1)
	dc := make(chan struct{}, 1)
	go cl.IssueCmd(ec, dc)
	select {
	case err := <-ec:
		return ret, err
	case <-time.After(time.Second * time.Duration(c.conf.Local.Timeout)):
		cl.CancelCmd()
		return ret, fmt.Errorf("Ping timed out")
	case <-dc:
	}
	resps := cl.GetResponses()
	var dw util.UUDecodingWriter
	for _, r := range resps {
		glog.Infof("Decoding and writing: %s", r.Bytes())
		_, err := r.WriteTo(&dw)
		if err != nil {
			return ret, err
		}
	}
	res, err := c.convertWarts(dw.Bytes())
	if err != nil {
		glog.Errorf("failed to decode ping with err: %v", err)
		return ret, err
	}
	err = decodeResponse(&res, &ret)
	if err != nil {
		glog.Exitf("Failed to decode traceroute with err: %v", err)
		return ret, err
	}
	c.increaseStats(sTime)
	glog.Infof("Traceroute done: %v", ret)
	return ret, nil
}

func (c *plControllerT) increaseStats(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests += 1
	c.time += time.Since(t)
}

func decodeResponse(res *[]byte, ret interface{}) error {
	return json.NewDecoder(bytes.NewReader(*res)).Decode(ret)
}

func (c *plControllerT) convertWarts(b []byte) ([]byte, error) {
	glog.Info("Converting Warts")
	res, err := util.ConvertBytes(c.sc.ScParserPath, b)
	if err != nil {
		glog.Errorf("Failed to converte bytes: %v", err)
		return []byte{}, err
	}
	glog.Infof("Results of converting: %s", res)
	return res, err
}

func (c *plControllerT) addSocket(sock scamper.Socket) {
	glog.Infof("Adding socket: %v, len: %d", sock, len(sock.IP()))
	c.rw.Lock()
	c.socks[sock.IP()] = sock
	glog.Infof("Sockets: %v", c.socks)
	c.rw.Unlock()
}

func (c *plControllerT) updateCanSpoof(ip int64) error {
	return c.db.UpdateCanSpoof(ip)
}

func (c *plControllerT) register(vp *dm.VantagePoint) error {
	return c.db.UpdateVp(vp)
}

func (c *plControllerT) updateVp(vp *dm.VantagePoint) error {
	return c.db.UpdateVp(vp)
}

func (c *plControllerT) getActiveVPs() ([]*dm.VantagePoint, error) {
	return c.db.GetActive()
}

func (c *plControllerT) getAllVPs() ([]*dm.VantagePoint, error) {
	return c.db.GetAll()
}

func (c *plControllerT) getRecordRouteVPs() ([]*dm.VantagePoint, error) {
	return c.db.GetRecordRoute()
}

func (c *plControllerT) getSpoofingVPs() ([]*dm.VantagePoint, error) {
	return c.db.GetSpoofers()
}

func (c *plControllerT) getTimeStampVPs() ([]*dm.VantagePoint, error) {
	return c.db.GetTimeStamps()
}

func (c *plControllerT) getVP(arg *dm.VPRequest) ([]*dm.VantagePoint, error) {
	hn := arg.Hostname
	if hn == "" {
		vp, err := c.db.GetVpByIp(arg.Ip)
		return []*dm.VantagePoint{vp}, err
	}
	vp, err := c.db.GetVpByHostname(hn)
	return []*dm.VantagePoint{vp}, err
}

func (c *plControllerT) getSocket(n string) (scamper.Socket, error) {
	glog.Infof("Getting socket for %s, len: %d", n, len(n))
	glog.Infof("Sockets: %v", c.socks)
	c.rw.RLock()
	defer c.rw.RUnlock()
	if sock, ok := c.socks[n]; ok {
		return sock, nil
	}
	glog.Errorf("Did not find socket for %s", n)
	return scamper.Socket{}, fmt.Errorf("Could not find socket: %s", n)
}

func (c *plControllerT) removeSocket(sock scamper.Socket) {
	glog.Infof("Removing socket: %v", sock)
	c.rw.Lock()
	delete(c.socks, sock.IP())
	c.rw.Unlock()
}

func Start(c Config, noScamp bool, db da.VantagePointProvider) chan error {
	glog.Info("Starting plcontroller")
	errChan := make(chan error, 2)
	if db == nil {
		errChan <- fmt.Errorf("Nill db in plController")
		return errChan
	}
	plController.db = db
	plController.reqCount = new(ReqCount)
	plController.socks = make(map[string]scamper.Socket, 10)
	var sc scamper.Config
	sc.Port = c.Scamper.Port
	sc.Path = c.Scamper.SockDir
	sc.ScPath = c.Scamper.BinPath
	sc.ScParserPath = c.Scamper.ConverterPath
	err := scamper.ParseConfig(sc)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)
		errChan <- err
		return errChan
	}
	plController.config = c
	plController.startTime = time.Now()
	plController.mp = mproc.New()
	plController.sc = sc
	plController.conf = c
	if !noScamp {
		plController.startScamperProc()
	}
	//Watch dir doesn't make the scamper dir if it doesn't exist so it's
	//best to call it after startScamperProc otherwise you'll send an error
	//and trigger any error logic in whatever code is using this
	plController.watchDir(sc.Path, errChan)
	var opts []grpc.ServerOption
	plController.server = grpc.NewServer(opts...)
	plc.RegisterPLControllerServer(plController.server, &plController)
	go plController.startRpc(errChan)
	return errChan
}

func (c *plControllerT) startRpc(eChan chan error) {
	var addr string
	if c.config.Local.AutoConnect {
		saddr, err := util.GetBindAddr()
		if err != nil {
			eChan <- err
			return
		}
		addr = fmt.Sprintf("%s:%d", saddr, 45000)
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
	glog.Infof("PLController started, listening on: %s", addr)
	err := c.server.Serve(l)
	if err != nil {
		eChan <- err
	}
}

func (c *plControllerT) startScamperProc() {
	sp := scamper.GetProc(c.sc.Path, c.sc.Port, c.sc.ScPath)
	plController.mp.ManageProcess(sp, true, 1000, handleScamperStop)
}

func HandleSig(s os.Signal) {
	plController.handleSig(s)
}

func (c *plControllerT) handleSig(s os.Signal) {
	glog.Infof("Got signale %v", s)
	if c.mp != nil {
		c.mp.KillAll()
	}
	if c.w != nil {
		c.w.Close()
	}
	c.removeAllVps()
	c.db.Close()
}
