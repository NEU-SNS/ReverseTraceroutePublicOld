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
package main

import (
	"flag"
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/plvp"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"os"
	"os/signal"
	"syscall"
)

var f plvp.Flags

func init() {
	flag.StringVar(&f.Port, "p", "55000",
		"The port the vp will bind to.")

	flag.StringVar(&f.Ip, "i", "127.0.0.1",
		"The IP that the vp will bind to.")

	flag.StringVar(&f.PType, "t", "tcp",
		"The protocol type the vp will use")

	flag.BoolVar(&f.CloseSocks, "D", false,
		"Determines if the standard file descriptors are closed.")

	flag.StringVar(&f.Url, "u", "localhost",
		"The url of the sc_remoted process")

	flag.StringVar(&f.ScPort, "P", "55000",
		"The destination port for scamper")

	flag.StringVar(&f.ScPath, "B", "/usr/local/bin/scamper",
		"Path to the scamper binary")
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		plvp.HandleSig(sig)
		os.Exit(1)
	}
}

func main() {
	go sigHandle()
	flag.Parse()
	defer glog.Flush()
	util.CloseStdFiles(f.CloseSocks)

	ipstr := fmt.Sprintf("%s:%s", f.Ip, f.Port)
	var sc scamper.ScamperConfig
	sc.Port = f.ScPort
	sc.ScPath = f.ScPath
	sc.Url = f.Url
	err := <-plvp.Start(f.PType, ipstr, sc)
	if err != nil {
		glog.Errorf("PLVP Start returned with error: %v", err)
		os.Exit(1)
	}
}
