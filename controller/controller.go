package controller

import (
	"github.com/NEU-SNS/ReverseTraceroute/lib"
	"github.com/golang/glog"
	"net"
)

type controllerT struct {
	port     int
	ip       string
	vps      map[string]*lib.Vantagepoint
	started  bool
	listener net.Listener
}

var controller controllerT

func Start(n, laddr string) {
	l, err := net.Listen(n, laddr)
	if err != nil {
		glog.Errorf("Controller failed to start. net: %s, addr: %s", n, laddr)
	}
	controller.listener = l
}
