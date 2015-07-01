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
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess/hyperdex"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
)

var f controller.Flags

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		glog.Infof("Got signal: %v", sig)
		controller.HandleSig(sig)
		exit(1)
	}
}

func init() {
	flag.StringVar(&f.Local.Addr, "a", ":35000",
		"The address that the controller will bind to.")
	flag.StringVar(&f.Local.Proto, "p", "tcp",
		"The protocol that the controller will use.")
	flag.BoolVar(&f.Local.CloseStdDesc, "D", false,
		"Determines if the sandard file descriptors are closed")
	flag.BoolVar(&f.Local.AutoConnect, "auto-connect", false,
		"Autoconnect to the eth0 IP, will use port 35000")
	flag.StringVar(&f.ConfigPath, "c", "",
		"Path to the config file.")
	flag.StringVar(&f.Local.PProfAddr, "pprof", "localhost:55555",
		"The port for pprof")
	flag.StringVar(&f.Local.CertFile, "cert-file", "cert.pem",
		"The path the the cert file for the the server")
	flag.StringVar(&f.Local.KeyFile, "key-file", "key.pem",
		"The path to the private key for the file")
}

func main() {
	go sigHandle()
	flag.Parse()
	defer glog.Flush()
	var conf controller.Config
	if f.ConfigPath != "" {
		err := config.ParseConfig(f.ConfigPath, &conf)
		if err != nil {
			glog.Errorf("Failed to parse config file: %s", f.ConfigPath)
			exit(1)
		}
	} else {
		conf.Local = f.Local
	}
	util.CloseStdFiles(f.Local.CloseStdDesc)
	util.StartPProf(conf.Local.PProfAddr)
	db, err := da.New(conf.Db)
	if err != nil {
		glog.Errorf("Failed to create db: %v", err)
		exit(1)
	}

	glog.Infof("Starting controller with config: %v", conf)
	err = <-controller.Start(conf, db, cache.New())
	if err != nil {
		glog.Errorf("Controller Start returned with error: %v", err)
		exit(1)
	}
}

func exit(status int) {
	glog.Flush()
	os.Exit(status)
}
