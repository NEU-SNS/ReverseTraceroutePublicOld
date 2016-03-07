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
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/prometheus/client_golang/prometheus"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"net/http"
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

var id = rand.Uint32()

func getName() string {
	name, err := os.Hostname()
	if err != nil {
		return fmt.Sprintf("plvp_%d", id)
	}
	r := strings.NewReplacer(".", "_", "-", "")
	return fmt.Sprintf("plvp_%s", r.Replace(name))
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
	monec    chan error
	monip    chan dm.Probe
	am       sync.Mutex // protect addr
	addr     string
	conn     *grpc.ClientConn
}

var plVantagepoint plVantagepointT

func (vp *plVantagepointT) handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	sip, e := pickIP(*vp.config.Local.Host)
	if e != nil {
		log.Errorf("Couldn't resolve host on restart")
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
	log.Infof("Got signal: %v", s)
	vp.stop()
}

func (vp *plVantagepointT) stop() {
	if vp.mp != nil {
		log.Infoln("Killing all processes")
		vp.mp.KillAll()
	}
	if vp.spoofmon != nil {
		vp.spoofmon.Quit()
	}
	if vp.conn != nil {
		vp.conn.Close()
	}
}

// HandleSig handles signals
func HandleSig(s os.Signal) {
	plVantagepoint.handleSig(s)
}

// The vp is dead if this method needs to return, so call stop() to clean up before returning
func (vp *plVantagepointT) run(c Config, ec chan error) {
	vp.config = c
	con := new(scamper.Config)
	con.ScPath = *c.Scamper.BinPath

	con.IP = *c.Scamper.Host
	con.Port = *c.Scamper.Port
	err := scamper.ParseConfig(*con)
	if err != nil {
		log.Errorf("Invalid scamper args: %v", err)
		vp.stop()
		ec <- err
		return
	}
	sip, err := pickIP(*c.Scamper.Host)
	if err != nil {
		log.Errorf("Could not resolve url: %s, with err: %v", *c.Local.Host, err)
		vp.stop()
		ec <- err
		return
	}
	vp.addr = sip
	vp.sc = *con
	vp.mp = mproc.New()
	vp.spoofmon = NewSpoofPingMonitor()
	monaddr, err := util.GetBindAddr()
	if err != nil {
		log.Errorf("Could not get bind addr: %v", err)
		vp.stop()
		ec <- err
		return
	}
	vp.monec = make(chan error, 1)
	vp.monip = make(chan dm.Probe, 100)
	go vp.spoofmon.Start(monaddr, plVantagepoint.monip, plVantagepoint.monec)
	go vp.monitorSpoofedPings(plVantagepoint.monip, plVantagepoint.monec)
	if *c.Local.StartScamp {
		plVantagepoint.startScamperProcs()
	}
}

func startHTTP(addr string) {
	for {
		log.Error(http.ListenAndServe(addr, nil))
	}
}

// Start a plvp with the given config
func Start(c Config) chan error {
	log.Info("Starting plvp with config: %v", c)
	http.Handle("/metrics", prometheus.Handler())
	go startHTTP(*c.Local.PProfAddr)
	errChan := make(chan error, 1)
	go plVantagepoint.run(c, errChan)
	return errChan

}

func (vp *plVantagepointT) sendSpoofs(probes []*dm.Probe) {
	log.Debug("Sending spoofed probes")
	if len(probes) == 0 {
		log.Debug("No Spoofs to send")
		return
	}
	if vp.conn == nil {
		_, srvs, err := net.LookupSRV("plcontroller", "tcp", "revtr.ccs.neu.edu")
		if err != nil {
			log.Error(err)
			return
		}
		creds, err := credentials.NewClientTLSFromFile(*vp.config.Local.RootCA, srvs[0].Target)
		if err != nil {
			log.Error(err)
			return
		}
		addr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
		log.Debugf("Dialing %s\n", addr)
		cc, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
		if err != nil {
			log.Errorf("Failed to send spoofs: %v", err)
			return
		}
		vp.conn = cc
	}
	log.Debug("About to send spoofed probes")
	client := plc.NewPLControllerClient(vp.conn)
	_, err := client.AcceptProbes(ctx.Background(), &dm.SpoofedProbes{Probes: probes})
	if err != nil {
		log.Errorf("Error sending probes: %v", err)
	}
	log.Info("Done sending spoofs")
}

func (vp *plVantagepointT) monitorSpoofedPings(probes chan dm.Probe, ec chan error) {
	var sprobes []*dm.Probe
	go func() {
		for {
			select {
			case probe := <-probes:
				log.Infof("Got IP from spoof monitor: %v", probe)
				spoofCounter.Inc()
				sprobes = append(sprobes, &probe)
			case err := <-ec:
				switch err {
				case ErrorNotICMPEcho, ErrorNonSpoofedProbe:
					continue
				}
				log.Errorf("Recieved error from spoof monitor: %v", err)
			case <-time.After(2 * time.Second):
				vp.sendSpoofs(sprobes)
				sprobes = make([]*dm.Probe, 0)
			}
		}
	}()
}

func pickIP(host string) (string, error) {

	log.Infof("Looking up addresses for %s", host)
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}

	log.Infof("Got IPs: %v", addrs)
	return addrs[rand.Intn(len(addrs))], nil
}

func (vp *plVantagepointT) startScamperProcs() {
	log.Info("Starting scamper procs")
	sp := scamper.GetVPProc(vp.sc.ScPath, vp.sc.IP, vp.sc.Port)
	vp.mp.ManageProcess(sp, true, 10000, vp.handleScamperStop)
}
