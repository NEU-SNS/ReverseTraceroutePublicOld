package main

import (
	"flag"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	"github.com/NEU-SNS/ReverseTraceroute/lib/scamper"
	"github.com/golang/glog"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	ps := scamper.GetProc("/tmp/scamper_sockets",
		"35000", "/usr/local/bin/sc_remoted")
	mt := scamper.GetMeasurementTool("/tmp/scamper_sockets")
	controller.Start("tcp", ":45000", mt, ps)
	err := controller.Accept()
	if err != nil {
		glog.Errorf("Controller: Listen returned error: %v", err)
	}
}
