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
	"github.com/NEU-SNS/ReverseTraceroute/lib/config"
	"github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess/hyperdex"
	"github.com/NEU-SNS/ReverseTraceroute/lib/plcontroller"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

var f plcontroller.Flags

func init() {
	flag.Int64Var(&f.Local.Timeout, "t", 60,
		"The default timeout used for measurement requests.")
	flag.StringVar(&f.Local.Addr, "a", ":45000",
		"The address that the controller will bind to.")
	flag.BoolVar(&f.Local.AutoConnect, "auto-connect", false,
		"Autoconnect to the eth0 IP, will use port 45000")
	flag.StringVar(&f.Local.Proto, "p", "tcp",
		"The protocol that the controller will use.")
	flag.BoolVar(&f.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed.")
	flag.StringVar(&f.Scamper.Port, "P", "55000",
		"Port that Scamper will use.")
	flag.StringVar(&f.Scamper.SockDir, "S", "/tmp/scamper_sockets",
		"Directory that scamper will use for its sockets")
	flag.StringVar(&f.Scamper.BinPath, "B", "/usr/local/bin/sc_remoted",
		"Path to the scamper binary")
	flag.StringVar(&f.Scamper.ConverterPath, "X", "/usr/local/bin/sc_warts2json",
		"Path for warts parser")
	flag.StringVar(&f.ConfigPath, "c", "",
		"Path to the config file")
	flag.StringVar(&f.Local.PProfAddr, "pprof", "localhost:55556",
		"The port for pprof")
	flag.StringVar(&f.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.StringVar(&f.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		glog.Infof("Got signal: %v", sig)
		plcontroller.HandleSig(sig)
		exit(1)
	}
}

func main() {
	go sigHandle()
	flag.Parse()
	defer glog.Flush()
	var conf plcontroller.Config

	if f.ConfigPath != "" {
		err := config.ParseConfig(f.ConfigPath, &conf)
		if err != nil {
			glog.Errorf("Failed to parse config file: %s", f.ConfigPath)
			exit(1)
		}
	} else {
		conf.Local = f.Local
		conf.Scamper = f.Scamper
	}
	util.CloseStdFiles(conf.Local.CloseStdDesc)
	util.StartPProf(conf.Local.PProfAddr)
	db, err := hdclient.NewVP(conf.Db)
	if err != nil {
		glog.Errorf("Failed to created db: %v", err)
		exit(1)
	}
	err = <-plcontroller.Start(conf, false, db)
	if err != nil {
		glog.Errorf("PLController Start returned with error: %v", err)
		exit(1)
	}
}

func exit(status int) {
	glog.Flush()
	os.Exit(status)
}
