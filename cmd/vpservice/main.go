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
	flag.Parse()
	svc := vpservice.NewRVPService()
	go sigHandle(svc)
	svc.LoadFromFile("./backup.txt")
	ln, err := net.Listen("tcp", "0.0.0.0:45000")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	serv := grpc.NewServer()
	pb.RegisterVPServiceServer(serv, vpservice.GRPCServ{VPService: svc})
	serv.Serve(ln)
}

func sigHandle(s *vpservice.RVPService) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for sig := range c {
		log.Infof("Got signal: %v", sig)
		s.StoreInFile("./backup.txt")
		os.Exit(1)
	}
}
