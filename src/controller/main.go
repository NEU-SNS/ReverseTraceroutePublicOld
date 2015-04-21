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
     * Neither the name of the University of Washington nor the
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
	"github.com/NEU-SNS/ReverseTraceroute/lib/controller"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess/testdataaccess"
	"github.com/golang/glog"
	"os"
)

var flags controller.Flags
var closeStdStreams bool

func init() {
	flag.StringVar(&flags.Port, "p", "35000",
		"The port that the controller will bind to.")

	flag.StringVar(&flags.Ip, "i", "127.0.0.1",
		"The IP that the controller will bind to.")

	flag.StringVar(&flags.PType, "t", "tcp",
		"Type protocol type the coltroller will use.")
	flag.BoolVar(&closeStdStreams, "D", false,
		"Determines if the sandard file descriptors are closed")
}

func main() {
	flag.Parse()
	if closeStdStreams {
		glog.Info("Closing standard file descripters")
		err := os.Stdin.Close()
		if err != nil {
			glog.Error("Failed to close Stdin")
			glog.Flush()
			os.Exit(1)
		}
		err = os.Stderr.Close()
		if err != nil {
			glog.Error("Failed to close Stderr")
			glog.Flush()
			os.Exit(1)
		}
		err = os.Stdout.Close()
		if err != nil {
			glog.Error("Failed to close Stdout")
			glog.Flush()
			os.Exit(1)
		}
	}

	ipstr := fmt.Sprintf("%s:%s", flags.Ip, flags.Port)
	err := <-controller.Start(flags.PType, ipstr, da.New())
	if err != nil {
		glog.Errorf("Controller Start returned with error: %v", err)
		glog.Flush()
		os.Exit(1)
	}
	glog.Flush()
}
