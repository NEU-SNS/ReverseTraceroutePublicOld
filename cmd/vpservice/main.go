package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
)

// Config is the config struct for the atlas
type Config struct {
	KeyFile  string
	CertFile string
	RootCA   string
}

func init() {
	config.SetEnvPrefix("ATLAS")
	config.AddConfigPath("./vpservice.config")
}

func main() {
	conf := Config{}
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	svc := vpservice.NewRVPService(conf.RootCA)
	go sigHandle(svc)
	svc.LoadFromFile("./backup.txt")
	ln, err := net.Listen("tcp", "0.0.0.0:45000")
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	creds, err := credentials.NewServerTLSFromFile(conf.CertFile, conf.KeyFile)
	if err != nil {
		log.Error(err)
		log.Error(conf)
		ln.Close()
		os.Exit(1)
	}
	serv := grpc.NewServer(grpc.Creds(creds))
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
