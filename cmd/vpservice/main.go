package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
)

func main() {
	go sigHandle()
	flag.Parse()
	svc := vpservice.NewRVPService()
	ln, err := net.Listen("tcp", "0.0.0.0:45000")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	serv := grpc.NewServer()
	pb.RegisterVPServiceServer(serv, vpservice.GRPCServ{svc})
	serv.Serve(ln)
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		log.Infof("Got signal: %v", sig)
		os.Exit(1)
	}
}
