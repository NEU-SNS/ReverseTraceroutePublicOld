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
package plvp

import (
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

type plVantagepointT struct {
	sc       scamper.Config
	spoofmon *SpoofPingMonitor
	dest     string
	mp       mproc.MProc
	config   Config
	mu       sync.Mutex
	lu       time.Time
	ips      []string
}

var plVantagepoint plVantagepointT

func (c *plVantagepointT) handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	sip, e := pickIp(c.config.Local.Host)
	if e != nil {
		glog.Errorf("Couldn't resolve host on restart")
		return true
	}
	c.sc.Ip = sip
	arg := fmt.Sprintf("%s:%s", sip, c.sc.Port)
	p.SetArg(scamper.ADDRINDEX, arg)
	switch err.(type) {
	default:
		return false
	case *os.PathError:
		return true
	}
}

func (c *plVantagepointT) handleSig(s os.Signal) {
	c.mp.KillAll()
}

func HandleSig(s os.Signal) {
	plVantagepoint.handleSig(s)
}

func Start(c Config) chan error {
	glog.Info("Starting plvp with config: %v", c)
	defer glog.Flush()
	errChan := make(chan error, 1)

	plVantagepoint.config = c

	con := new(scamper.Config)
	con.ScPath = c.Scamper.BinPath
	sip, err := pickIp(c.Scamper.Host)
	if err != nil {
		glog.Errorf("Could not resolve url: %s, with err: %v", c.Local.Host, err)
		errChan <- err
		return errChan
	}
	con.Ip = sip
	con.Port = c.Scamper.Port
	err = scamper.ParseConfig(*con)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)
		errChan <- err
		return errChan
	}
	plVantagepoint.sc = *con
	plVantagepoint.mp = mproc.New()
	plVantagepoint.spoofmon = &SpoofPingMonitor{}
	monaddr, err := util.GetBindAddr()
	if err != nil {
		glog.Errorf("Could not get bind addr: %v", err)
		errChan <- err
		return errChan
	}
	monec := make(chan error, 1)
	monip := make(chan net.IP, 1)
	go plVantagepoint.spoofmon.Start(monaddr, monip, monec)
	if c.Local.StartScamp {
		plVantagepoint.startScamperProcs()
	}

	go func() {
		for {
			select {
			case ip := <-monip:
				glog.Infof("Got IP from spoof monitor: %d", ip)
			case err := <-monec:
				glog.Errorf("Recieved error from spoof monitor: %v", err)
			}
		}
	}()
	return errChan
}

func pickIp(host string) (string, error) {
	plVantagepoint.mu.Lock()
	defer plVantagepoint.mu.Unlock()

	if len(plVantagepoint.ips) == 0 || time.Since(plVantagepoint.lu) > time.Minute*5 {
		glog.Infof("Looking up addresses for %s", host)
		addrs, err := net.LookupHost(host)
		if err != nil {
			return "", err
		}

		glog.Infof("Got IPs: %v", addrs)
		plVantagepoint.ips = addrs
		plVantagepoint.lu = time.Now()
	}
	rand.Seed(time.Now().UnixNano())
	return plVantagepoint.ips[rand.Intn(len(plVantagepoint.ips))], nil
}

func (c *plVantagepointT) startScamperProcs() {
	glog.Info("Starting scamper procs")
	sp := scamper.GetVPProc(c.sc.ScPath, c.sc.Ip, c.sc.Port)
	c.mp.ManageProcess(sp, true, 10000, c.handleScamperStop)
}
