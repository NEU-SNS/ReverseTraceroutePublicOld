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
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"net"
	"os"
	"strings"
)

type plVantagepointT struct {
	ip       net.IP
	port     int
	hostname string
	sc       []scamper.Config
	dest     string
	mp       mproc.MProc
}

var plVantagepoint plVantagepointT

func handleScamperStop(err error, ps *os.ProcessState) bool {
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
	glog.Info("Starting plvp")
	defer glog.Flush()
	errChan := make(chan error, 1)

	port, ip, err := util.ParseAddrArg(c.Local.Addr)
	if err != nil {
		glog.Error("Failed to parse addr string")
		errChan <- err
		return errChan
	}

	plVantagepoint.sc = make([]scamper.Config, len(c.Scamper.Addrs))

	for i, addr := range c.Scamper.Addrs {
		con := new(scamper.Config)
		split := strings.Split(addr, ":")
		con.Url = split[util.IP]
		con.Port = split[util.PORT]
		con.ScPath = c.Scamper.BinPath
		err = scamper.ParseConfig(*con)
		if err != nil {
			glog.Errorf("Invalid scamper args: %v", err)
			errChan <- err
			return errChan
		}
		plVantagepoint.sc[i] = *con
	}
	plVantagepoint.port = port
	plVantagepoint.ip = ip
	plVantagepoint.mp = mproc.New()
	plVantagepoint.startScamperProcs()
	return errChan
}

func (c *plVantagepointT) startScamperProcs() {
	glog.Info("Starting scamper procs")
	for _, sc := range c.sc {

		sp := scamper.GetVPProc(sc.ScPath, sc.Url, sc.Port)
		c.mp.ManageProcess(sp, true, 10, handleScamperStop)
	}
}