package main

import (
	"flag"
	"github.com/NEU-SNS/ReverseTraceroute/controller"
	"github.com/golang/glog"
	"net"
	"net/rpc/jsonrpc"
	"runtime"
)

const (
	PING       = "ControllerApi.Ping"
	TRACEROUTE = "ControllerApi.Traceroute"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	controller.Start("tcp", "localhost:45000")
	conn, err := net.Dial("tcp", "localhost:45000")

	if err != nil {
		panic(err)
	}
	defer conn.Close()

	c := jsonrpc.NewClient(conn)
	var result int
	glog.Info("Calling CApi")
	err = c.Call(PING, 5, &result)
	glog.Infof("Done with remote call")
	if err != nil {
		panic(err)
	}
	glog.Infof("Results %d", result)
	glog.Flush()
}
