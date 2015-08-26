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

//Package plcontroller is the library for creating a planet-lab controller
package plcontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"gopkg.in/fsnotify.v1"

	"net/http"
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
		return fmt.Sprintf("plcontroller_%d", id)
	}
	r := strings.NewReplacer(".", "_", "-", "")
	return fmt.Sprintf("plcontroller_%s", r.Replace(name))
}

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(rpcCounter)
	prometheus.MustRegister(timeoutCounter)
	prometheus.MustRegister(errorCounter)
}

type plControllerT struct {
	spid     int
	server   *grpc.Server
	config   Config
	sc       scamper.Config
	mp       mproc.MProc
	db       da.VPProvider
	w        *fsnotify.Watcher
	client   Client
	spoofs   *spoofMap
	ip       uint32
	shutdown chan struct{}
}

// Client is the measurment client interface
// TODO: Remove interface dependency on scamper
type Client interface {
	AddSocket(*scamper.Socket)
	RemoveSocket(string)
	GetSocket(string) (*scamper.Socket, error)
	RemoveMeasurement(string, uint32) error
	DoMeasurement(string, interface{}) (<-chan scamper.Response, uint32, error)
	GetAllSockets() <-chan *scamper.Socket
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

func (c *plControllerT) recSpoof(rs *dm.Spoof) (*dm.NotifyRecSpoofResponse, error) {
	resp := &dm.NotifyRecSpoofResponse{}
	err := c.spoofs.Register(*rs)
	return resp, err
}

func (c *plControllerT) runPing(pa *dm.PingMeasurement) (dm.Ping, error) {
	glog.V(2).Infof("Running ping for: %v", pa)
	timeout := pa.Timeout
	if timeout == 0 {
		timeout = *c.config.Local.Timeout
	}
	ret := dm.Ping{}

	resp, id, err := c.client.DoMeasurement(pa.Src, pa)
	if err != nil {
		return ret, err
	}
	rpcCounter.Inc()
	select {
	case r := <-resp:
		err := decodeResponse(r.Bytes(), &ret)
		if err != nil {
			errorCounter.Inc()
			return ret, fmt.Errorf("Could not decode ping response: %v, resp: %s", err, r.Bytes())
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		timeoutCounter.Inc()
		c.client.RemoveMeasurement(pa.Src, id)
		return ret, fmt.Errorf("Ping timed out")
	}
	return ret, nil
}

func (c *plControllerT) acceptProbe(probe *dm.Probe) error {
	return c.spoofs.Receive(*probe)
}

func (c *plControllerT) runTraceroute(ta *dm.TracerouteMeasurement) (dm.Traceroute, error) {
	glog.Infof("Running traceroute for: %v", ta)
	timeout := ta.Timeout
	if timeout == 0 {
		timeout = *c.config.Local.Timeout
	}
	ret := dm.Traceroute{}

	resp, id, err := c.client.DoMeasurement(ta.Src, ta)
	if err != nil {
		return ret, err
	}
	rpcCounter.Inc()
	select {
	case r := <-resp:
		err := decodeResponse(r.Bytes(), &ret)
		if err != nil {
			errorCounter.Inc()
			return ret, fmt.Errorf("Could not decode traceroute response: %v", err)
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		timeoutCounter.Inc()
		c.client.RemoveMeasurement(ta.Src, id)
		return ret, fmt.Errorf("Traceroute timed out")
	}
	return ret, nil
}

func decodeResponse(res []byte, ret interface{}) error {
	return json.NewDecoder(bytes.NewReader(res)).Decode(ret)
}

func convertWarts(path string, b []byte) ([]byte, error) {
	glog.Info("Converting Warts")
	res, err := util.ConvertBytes(path, b)
	if err != nil {
		glog.Errorf("Failed to converte bytes: %v", err)
		return []byte{}, err
	}
	glog.Infof("Results of converting: %s", res)
	return res, err
}

// When this returns the server is essentially dead, so call stop before any return
func (c *plControllerT) run(ec chan error, con Config, noScamp bool, db da.VPProvider, cl Client, s Sender) {
	defer glog.Flush()
	if db == nil {
		c.stop()
		ec <- fmt.Errorf("Nil db in plController")
		return
	}
	var sc scamper.Config
	sc.Port = *con.Scamper.Port
	sc.Path = *con.Scamper.SockDir
	sc.ScPath = *con.Scamper.BinPath
	sc.ScParserPath = *con.Scamper.ConverterPath
	err := scamper.ParseConfig(sc)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)

		c.stop()
		ec <- err
		return
	}
	ips, err := util.GetBindAddr()
	if err != nil {
		glog.Errorf("Failed to get bind address: %v", err)
		c.stop()
		ec <- err
		return
	}
	ip, err := util.IPStringToInt32(ips)
	if err != nil {
		glog.Errorf("Failed to convert ip string: %v", err)
		c.stop()
		ec <- err
		return
	}

	c.db = db
	c.ip = ip
	c.shutdown = make(chan struct{})
	c.spoofs = newSpoofMap(s)
	c.config = con
	c.mp = mproc.New()
	c.sc = sc
	if !noScamp {
		c.startScamperProc()
	}
	c.client = cl
	c.watchDir(sc.Path, ec)
	c.server = grpc.NewServer()
	plc.RegisterPLControllerServer(plController.server, c)
	go c.startRPC(ec)
	/*
		go func() {
			glog.Info("Starting VP monitoring")
			for {
				select {
				case <-c.shutdown:
					return
				case <-time.After(time.Minute * 2):
					glog.Info("Checking VPs....")
					vps, err := c.db.GetVPs()
					if err != nil {
						glog.Errorf("Failed to get VPs: %v", err)
						return
					}
					err = maintainVPs(
						vps,
						*c.config.Local.PLUName,
						*c.config.Local.SSHKeyPath,
						*c.config.Local.UpdateUrl,
						c.db,
						c.shutdown,
					)
					if err != nil {
						glog.Errorf("Failed to maintain VPS: %v", err)
						return
					}
				}
			}
		}()
	*/
}

func startHttp(addr string) {
	for {
		glog.Error(http.ListenAndServe(addr, nil))
	}
}

// Start starts a plcontroller with the given configuration
func Start(c Config, noScamp bool, db da.VPProvider, cl Client, s Sender) chan error {
	glog.Info("Starting plcontroller")
	http.Handle("/metrics", prometheus.Handler())
	go startHttp(*c.Local.PProfAddr)
	errChan := make(chan error, 2)
	go plController.run(errChan, c, noScamp, db, cl, s)
	return errChan
}

func (c *plControllerT) startRPC(eChan chan error) {
	addr := fmt.Sprintf("%s:%d", *c.config.Local.Addr,
		*c.config.Local.Port)
	glog.Infof("Conecting to: %s", addr)
	l, e := net.Listen("tcp", addr)
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
	c.mp.ManageProcess(sp, true, 1000, handleScamperStop)
}

// HandleSig allows the plController to react appropriately to signals
func HandleSig(s os.Signal) {
	plController.handleSig(s)
}

func (c *plControllerT) stop() {
	if c.shutdown != nil {
		close(c.shutdown)
	}
	if c.w != nil {
		c.w.Close()
	}
	if c.mp != nil {
		c.mp.KillAll()
	}
	if c.db != nil {
		c.removeAllVps()
		c.db.Close()
	}
	if c.spoofs != nil {
		c.spoofs.Quit()
	}
	// Wait 5 seconds... I think sc_remoted needs time to properly clean-up
	<-time.After(time.Second * 5)
	cleanDir(*c.config.Scamper.SockDir)
}

func (c *plControllerT) handleSig(s os.Signal) {
	glog.Infof("Got signal: %v", s)
	c.stop()
}
