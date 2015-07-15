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
	"sync"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type plVantagepointT struct {
	sc       scamper.Config
	spoofmon *SpoofPingMonitor
	mp       mproc.MProc
	config   Config
	mu       sync.Mutex
	lu       time.Time
	plc      *plClient
	monec    chan error
	monip    chan dm.Probe
	am       sync.Mutex // protect addr
	addr     string
}

var plVantagepoint plVantagepointT

func (c *plVantagepointT) handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	sip, e := pickIP(c.config.Local.Host)
	if e != nil {
		glog.Errorf("Couldn't resolve host on restart")
		return true
	}
	c.sc.IP = sip
	arg := fmt.Sprintf("%s:%s", sip, c.sc.Port)
	c.am.Lock()
	c.addr = fmt.Sprintf("%s:%s", sip, c.config.Local.Port)
	c.am.Unlock()
	p.SetArg(scamper.ADDRINDEX, arg)
	c.spoofmon.Quit()
	c.spoofmon = NewSpoofPingMonitor()
	c.spoofmon.Start(arg, c.monip, c.monec)
	switch err.(type) {
	default:
		return false
	case *os.PathError:
		return true
	}
}

func (c *plVantagepointT) handleSig(s os.Signal) {
	c.mp.KillAll()
	c.spoofmon.Quit()
}

// HandleSig handles signals
func HandleSig(s os.Signal) {
	plVantagepoint.handleSig(s)
}

// Start a plvp with the given config
func Start(c Config) chan error {
	glog.Info("Starting plvp with config: %v", c)
	defer glog.Flush()
	errChan := make(chan error, 1)

	plVantagepoint.config = c

	con := new(scamper.Config)
	con.ScPath = c.Scamper.BinPath
	sip, err := pickIP(c.Scamper.Host)
	if err != nil {
		glog.Errorf("Could not resolve url: %s, with err: %v", c.Local.Host, err)
		errChan <- err
		return errChan
	}
	con.IP = sip
	con.Port = c.Scamper.Port
	err = scamper.ParseConfig(*con)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)
		errChan <- err
		return errChan
	}
	plVantagepoint.addr = sip
	plVantagepoint.sc = *con
	plVantagepoint.mp = mproc.New()
	plVantagepoint.spoofmon = NewSpoofPingMonitor()
	plVantagepoint.plc = &plClient{}
	monaddr, err := util.GetBindAddr()
	if err != nil {
		glog.Errorf("Could not get bind addr: %v", err)
		errChan <- err
		return errChan
	}
	plVantagepoint.monec = make(chan error, 1)
	plVantagepoint.monip = make(chan dm.Probe, 1)
	go plVantagepoint.spoofmon.Start(monaddr, plVantagepoint.monip, plVantagepoint.monec)
	go plVantagepoint.monitorSpoofedPings(plVantagepoint.monip, plVantagepoint.monec)
	if c.Local.StartScamp {
		plVantagepoint.startScamperProcs()
	}
	return errChan
}

func (c *plVantagepointT) sendSpoofs(probes []*dm.Probe) {
	if len(probes) == 0 {
		return
	}
	c.am.Lock()
	ip := c.addr
	c.am.Unlock()
	addr := fmt.Sprintf("%s:%s", ip, c.config.Local.Port)
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

func (c *plVantagepointT) monitorSpoofedPings(probes chan dm.Probe, ec chan error) {
	sprobes := make([]*dm.Probe, 0)
	go func() {
		for {
			select {
			case probe := <-probes:
				glog.Infof("Got IP from spoof monitor: %v", probe)
				sprobes = append(sprobes, &probe)
			case err := <-ec:
				glog.Errorf("Recieved error from spoof monitor: %v", err)
			case <-time.After(2 * time.Second):
				c.sendSpoofs(sprobes)
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
