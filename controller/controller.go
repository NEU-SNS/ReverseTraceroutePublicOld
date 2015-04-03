package controller

import (
	vp "github.com/NEU-SNS/ReverseTraceroute/lib/vp"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	"github.com/golang/glog"
	"net"
)

type MeasurementTool interface {
	TraceRoute()
	Ping()
	RRPing()
	TSPing()
	SpoofTr()
}

type controllerT struct {
	port          int
	ip            string
	vps           map[string]*vp.Vantagepoint
	started       bool
	listener      net.Listener
	mt            MeasurementTool
	manager       mproc.MProc
	procId        int
	init          bool
	managedProcId int
}

var controller controllerT

func initController(mt MeasurementTool) {
	if controller.init {
		return
	}
	controller.vps = make(map[string]*vp.Vantagepoint, 10)
	controller.manager = mproc.New()
	controller.mt = mt
	controller.init = true
}

func Start(n, laddr string, mt MeasurementTool, procs *proc.Process) {
	initController(mt)
	if controller.started {
		return
	}
	id, err := controller.manager.ManageProcess(procs, true)
	if err != nil {
		glog.Fatalf("Controller: manage process failed: %v", err)
	}
	controller.managedProcId = id

	l, err := net.Listen(n, laddr)
	if err != nil {
		glog.Fatal("Controller failed to start. net: %s, addr: %s, error: ",
			n, laddr, err)
	}
	glog.Infof("Controller started, listening on %s", laddr)
	controller.listener = l
	controller.started = true
	for {
		l.Accept()
	}
}
