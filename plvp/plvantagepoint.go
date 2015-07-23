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

// Package plvp is the library for creating a vantage poing on a planet-lab node
package plvp

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"

	"net/http"
	_ "net/http/pprof"
)

var (
	procCollector = prometheus.NewProcessCollectorPIDFn(func() (int, error) {
		return os.Getpid(), nil
	}, getName())
	spoofCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: getName(),
		Subsystem: "spoof",
		Name:      "count",
		Help:      "Count of the spoofed probes received",
	})
)

var id uint32 = rand.Uint32()

func getName() string {
	name, err := os.Hostname()
	if err != nil {
		return fmt.Sprintf("plvp_%d", id)
	}
	return fmt.Sprintf("plvp_%s", strings.Replace(name, ".", "_", -1))
}

func init() {
	prometheus.MustRegister(procCollector)
	prometheus.MustRegister(spoofCounter)
}

type plVantagepointT struct {
	sc       scamper.Config
	spoofmon *SpoofPingMonitor
	mp       mproc.MProc
	config   Config
	mu       sync.Mutex
	plc      *plClient
	monec    chan error
	monip    chan dm.Probe
	am       sync.Mutex // protect addr
	addr     string
}

var plVantagepoint plVantagepointT

func (vp *plVantagepointT) handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	sip, e := pickIP(*vp.config.Local.Host)
	if e != nil {
		glog.Errorf("Couldn't resolve host on restart")
		return true
	}
	vp.sc.IP = sip
	arg := fmt.Sprintf("%s:%s", sip, vp.sc.Port)
	vp.am.Lock()
	vp.addr = fmt.Sprintf("%s:%d", sip, *vp.config.Local.Port)
	vp.am.Unlock()
	p.SetArg(scamper.ADDRINDEX, arg)
	switch err.(type) {
	default:
		return false
	case *os.PathError:
		return true
	}
}

func (vp *plVantagepointT) handleSig(s os.Signal) {
	glog.Infof("Got signal: %v", s)
	vp.stop()
}

func (vp *plVantagepointT) stop() {
	if vp.mp != nil {
		vp.mp.KillAll()
	}
	if vp.spoofmon != nil {
		vp.spoofmon.Quit()
	}
}

// HandleSig handles signals
func HandleSig(s os.Signal) {
	plVantagepoint.handleSig(s)
}

// The vp is dead if this method needs to return, so call stop() to clean up before returning
func (vp *plVantagepointT) run(c Config, ec chan error) {
	vp.config = c
	defer glog.Flush()
	con := new(scamper.Config)
	con.ScPath = *c.Scamper.BinPath
	sip, err := pickIP(*c.Scamper.Host)
	if err != nil {
		glog.Errorf("Could not resolve url: %s, with err: %v", *c.Local.Host, err)
		vp.stop()
		ec <- err
		return
	}
	con.IP = sip
	con.Port = *c.Scamper.Port
	err = scamper.ParseConfig(*con)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)
		vp.stop()
		ec <- err
		return
	}
	plVantagepoint.addr = sip
	plVantagepoint.sc = *con
	plVantagepoint.mp = mproc.New()
	plVantagepoint.spoofmon = NewSpoofPingMonitor()
	plVantagepoint.plc = &plClient{}
	monaddr, err := util.GetBindAddr()
	if err != nil {
		glog.Errorf("Could not get bind addr: %v", err)
		vp.stop()
		ec <- err
		return
	}
	plVantagepoint.monec = make(chan error, 1)
	plVantagepoint.monip = make(chan dm.Probe, 1)
	go plVantagepoint.spoofmon.Start(monaddr, plVantagepoint.monip, plVantagepoint.monec)
	go plVantagepoint.monitorSpoofedPings(plVantagepoint.monip, plVantagepoint.monec)
	if *c.Local.StartScamp {
		plVantagepoint.startScamperProcs()
	}
}

func startHttp(addr string) {
	for {
		glog.Error(http.ListenAndServe(addr, nil))
	}
}

// Start a plvp with the given config
func Start(c Config) chan error {
	glog.Info("Starting plvp with config: %v", c)
	http.Handle("/metrics", prometheus.Handler())
	go startHttp(*c.Local.PProfAddr)
	errChan := make(chan error, 1)
	go plVantagepoint.run(c, errChan)
	return errChan

}

func (vp *plVantagepointT) sendSpoofs(probes []*dm.Probe) {
	if len(probes) == 0 {
		return
	}
	vp.am.Lock()
	ip := vp.addr
	vp.am.Unlock()
	addr := fmt.Sprintf("%s:%s", ip, vp.config.Local.Port)
	cc, err := grpc.Dial(addr, grpc.WithTimeout(2*time.Second))
	if err != nil {
		glog.Errorf("Failed to send spoofs: %v", err)
		return
	}
	client := plc.NewPLControllerClient(cc)
	_, err = client.AcceptProbes(ctx.Background(), &dm.SpoofedProbes{Probes: probes})
	if err != nil {
		glog.Errorf("Error sending probes: %v", err)
	}
}

func (vp *plVantagepointT) monitorSpoofedPings(probes chan dm.Probe, ec chan error) {
	sprobes := make([]*dm.Probe, 0)
	go func() {
		for {
			select {
			case probe := <-probes:
				glog.Infof("Got IP from spoof monitor: %v", probe)
				spoofCounter.Inc()
				sprobes = append(sprobes, &probe)
			case err := <-ec:
				switch err {
				case ErrorNotICMPEcho, ErrorNonSpoofedProbe:
					continue
				}
				glog.Errorf("Recieved error from spoof monitor: %v", err)
			case <-time.After(2 * time.Second):
				vp.sendSpoofs(sprobes)
				sprobes = make([]*dm.Probe, 0)
			}
		}
	}()
}

func pickIP(host string) (string, error) {

	glog.Infof("Looking up addresses for %s", host)
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}

	glog.Infof("Got IPs: %v", addrs)
	return addrs[rand.Intn(len(addrs))], nil
}

func (c *plVantagepointT) startScamperProcs() {
	glog.Info("Starting scamper procs")
	sp := scamper.GetVPProc(c.sc.ScPath, c.sc.IP, c.sc.Port)
	c.mp.ManageProcess(sp, true, 10000, c.handleScamperStop)
}
