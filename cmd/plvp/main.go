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

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/plvp"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
)

var (
	defaultConfig = "./plvp.config"
	configPath    string
)

var conf plvp.Config = plvp.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	if configPath == "" {
		config.AddConfigPath(defaultConfig)
	} else {
		config.AddConfigPath(configPath)
	}

	flag.StringVar(conf.Local.Addr, "a", ":65000",
		"The address to run the local service on")
	flag.BoolVar(conf.Local.CloseStdDesc, "d", false,
		"Close std file descripters")
	flag.BoolVar(conf.Local.AutoConnect, "auto-connect", false,
		"Autoconnect to 0.0.0.0 and will use port 55000")
	flag.StringVar(conf.Local.PProfAddr, "pprof-addr", ":55557",
		"The address to use for pperf")
	flag.StringVar(conf.Local.Host, "host", "plcontroller.revtr.ccs.neu.edu",
		"The url for the plcontroller service")
	flag.IntVar(conf.Local.Port, "p", 4380,
		"The port the controller service is listening on")
	flag.BoolVar(conf.Local.StartScamp, "start-scamper", true,
		"Determines if scamper starts or not.")
	flag.StringVar(conf.Scamper.BinPath, "b", "/usr/local/bin/scamper",
		"The path to the scamper binary")
	flag.StringVar(conf.Scamper.Port, "scamper-port", "4381",
		"The port scamper will try to connect to.")
	flag.StringVar(conf.Scamper.Host, "scamper-host", "plcontroller.revtr.ccs.neu.edu",
		"The host that the sc_remoted process is running, should most likely match the host arg")
}

func main() {
	go sigHandle()
	defer glog.Flush()
	var parseConf plvp.Config
	err := config.Parse(flag.CommandLine, &parseConf)
	if err != nil {
		glog.Exitf("Failed to parse config: %v", err)
		exit(1)
	}
	util.CloseStdFiles(*conf.Local.CloseStdDesc)
	err = <-plvp.Start(conf)
	if err != nil {
		glog.Errorf("PLVP Start returned with error: %v", err)
		exit(1)
	}
}

func exit(status int) {
	glog.Flush()
	os.Exit(status)
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		glog.Infof("Got signal: %v", sig)
		plvp.HandleSig(sig)
		exit(1)
	}
}
