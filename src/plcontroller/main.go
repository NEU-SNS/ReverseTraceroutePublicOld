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
	"github.com/NEU-SNS/ReverseTraceroute/lib/plcontroller"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"os"
	"os/signal"
	"syscall"
)

var flags plcontroller.Flags

func init() {
	flag.StringVar(&flags.Port, "p", "45000",
		"The port that the controller will bind to.")

	flag.StringVar(&flags.Ip, "i", "127.0.0.1",
		"The IP that the controller will bind to.")

	flag.StringVar(&flags.PType, "t", "tcp",
		"Type protocol type the coltroller will use.")

	flag.BoolVar(&flags.CloseSocks, "D", false,
		"Determines if the sandard file descriptors are closed.")

	flag.StringVar(&flags.SPort, "P", "55000",
		"Port that Scamper will use.")

	flag.StringVar(&flags.SockPath, "S", "/tmp/scamper_sockets",
		"Directory that scamper will use for its sockets")

	flag.StringVar(&flags.ScPath, "B", "/usr/local/bin/sc_remoted",
		"Path to the scamper binary")
	flag.StringVar(&flags.ScParserPath, "X", "/usr/local/bin/sc_warts2json",
		"Path for warts parser")
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for sig := range c {
		plcontroller.HandleSig(sig)
		os.Exit(1)
	}
}

func main() {
	go sigHandle()
	flag.Parse()
	defer glog.Flush()
	util.CloseStdFiles(flags.CloseSocks)

	ipstr := fmt.Sprintf("%s:%s", flags.Ip, flags.Port)
	var sa scamper.ScamperConfig
	sa.Port = flags.SPort
	sa.Path = flags.SockPath
	sa.ScPath = flags.ScPath
	sa.ScParserPath = flags.ScParserPath
	err := <-plcontroller.Start(flags.PType, ipstr, sa)
	if err != nil {
		glog.Errorf("PLController Start returned with error: %v", err)
		os.Exit(1)
	}
}
