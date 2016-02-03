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
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/spoofmap"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/warts"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/fsnotify.v1"

	"net/http"
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
	server   *grpc.Server
	config   Config
	sc       scamper.Config
	mp       mproc.MProc
	db       *da.DataAccess
	w        *fsnotify.Watcher
	client   Client
	spoofs   *spoofmap.SpoofMap
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
	saddr, _ := util.Int32ToIPString(rs.Sip)
	dummy := &dm.PingMeasurement{
		Src:     rs.Ip,
		Dst:     rs.Dst,
		Spoof:   true,
		SAddr:   saddr,
		Count:   "1",
		Timeout: 2,
		Ttl:     "1",
	}
	src, _ := util.Int32ToIPString(rs.Sip)
	c.client.DoMeasurement(src, dummy)
	err := c.spoofs.Register(*rs)
	return resp, err
}

func (c *plControllerT) runPing(ctx context.Context, pa *dm.PingMeasurement) (dm.Ping, error) {
	log.Debugf("Running ping for: %v", pa)
	timeout := pa.Timeout
	if timeout == 0 {
		timeout = *c.config.Local.Timeout
	}
	src, err := util.Int32ToIPString(pa.Src)
	if err != nil {
		return dm.Ping{}, err
	}
	resp, id, err := c.client.DoMeasurement(src, pa)
	if err != nil {
		return dm.Ping{}, err
	}
	rpcCounter.Inc()
	select {
	case r := <-resp:
		switch t := r.Ret.(type) {
		case warts.Ping:
			return dm.ConvertPing(t), nil
		default:
			errorCounter.Inc()
			return dm.Ping{}, fmt.Errorf("Wrong type in ping response")
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		timeoutCounter.Inc()
		src, _ := util.Int32ToIPString(pa.Src)
		c.client.RemoveMeasurement(src, id)
		return dm.Ping{}, fmt.Errorf("Ping timed out")
	case <-ctx.Done():
		return dm.Ping{}, ctx.Err()
	}
}

func (c *plControllerT) acceptProbe(probe *dm.Probe) error {
	return c.spoofs.Receive(probe)
}

func (c *plControllerT) runTraceroute(ctx context.Context, ta *dm.TracerouteMeasurement) (dm.Traceroute, error) {
	timeout := ta.Timeout
	if timeout == 0 {
		timeout = *c.config.Local.Timeout
	}

	src, err := util.Int32ToIPString(ta.Src)
	if err != nil {
		return dm.Traceroute{}, err
	}
	resp, id, err := c.client.DoMeasurement(src, ta)
	if err != nil {
		return dm.Traceroute{}, err
	}
	rpcCounter.Inc()
	select {
	case r := <-resp:
		switch t := r.Ret.(type) {
		case warts.Traceroute:
			return dm.ConvertTraceroute(t), nil
		default:
			errorCounter.Inc()
			return dm.Traceroute{}, fmt.Errorf("Wrong type in traceroute response")
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		timeoutCounter.Inc()
		src, _ := util.Int32ToIPString(ta.Src)
		c.client.RemoveMeasurement(src, id)
		return dm.Traceroute{}, fmt.Errorf("Traceroute timed out")
	case <-ctx.Done():
		return dm.Traceroute{}, ctx.Err()
	}
}

func convertWarts(path string, b []byte) ([]byte, error) {
	res, err := util.ConvertBytes(path, b)
	if err != nil {
		log.Errorf("Failed to converte bytes: %v", err)
		return []byte{}, err
	}
	return res, err
}

// When this returns the server is essentially dead, so call stop before any return
func (c *plControllerT) run(ec chan error, con Config, noScamp bool, db *da.DataAccess, cl Client, s spoofmap.Sender) {
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
		log.Errorf("Invalid scamper args: %v", err)

		c.stop()
		ec <- err
		return
	}
	ips, err := util.GetBindAddr()
	if err != nil {
		log.Errorf("Failed to get bind address: %v", err)
		c.stop()
		ec <- err
		return
	}
	ip, err := util.IPStringToInt32(ips)
	if err != nil {
		log.Errorf("Failed to convert ip string: %v", err)
		c.stop()
		ec <- err
		return
	}
	c.db = db
	c.ip = ip
	c.shutdown = make(chan struct{})
	c.spoofs = spoofmap.New(s)
	c.config = con
	c.mp = mproc.New()
	c.sc = sc
	if !noScamp {
		c.startScamperProc()
	}
	c.client = cl
	c.watchDir(sc.Path, ec)
	creds, err := credentials.NewServerTLSFromFile(*con.Local.CertFile, *con.Local.KeyFile)
	if err != nil {
		log.Error(err)
		c.stop()
		ec <- err
		return
	}
	c.server = grpc.NewServer(grpc.Creds(creds))
	plc.RegisterPLControllerServer(c.server, c)
	go c.startRPC(ec)
	go c.maintain()
}

func (c *plControllerT) maintain() {
	for {
		select {
		case <-c.shutdown:
			return
		case <-time.After(time.Minute * 5):
			vps, err := c.db.GetVPs()
			if err != nil {
				log.Errorf("Failed to get VPs: %v", err)
				return
			}
			err = maintainVPs(
				vps,
				*c.config.Local.PLUName,
				*c.config.Local.SSHKeyPath,
				*c.config.Local.UpdateURL,
				c.db,
				c.shutdown,
			)
			if err != nil {
				log.Errorf("Failed to maintain VPS: %v", err)
				return
			}
		}
	}
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

// Start starts a plcontroller with the given configuration
func Start(c Config, noScamp bool, db *da.DataAccess, cl Client, s spoofmap.Sender) chan error {
	http.Handle("/metrics", prometheus.Handler())
	go startHTTP(*c.Local.PProfAddr)
	errChan := make(chan error, 2)
	go plController.run(errChan, c, noScamp, db, cl, s)
	return errChan
}

func (c *plControllerT) startRPC(eChan chan error) {
	addr := fmt.Sprintf("%s:%d", *c.config.Local.Addr,
		*c.config.Local.Port)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
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
		c.mp.IntAll()
	}
	c.removeAllVps()
	c.db.Close()
	if c.spoofs != nil {
		c.spoofs.Quit()
	}
	// Wait 5 seconds... I think sc_remoted needs time to properly clean-up
	<-time.After(time.Second * 5)
}

func (c *plControllerT) handleSig(s os.Signal) {
	c.stop()
}
