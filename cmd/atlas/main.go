package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/NEU-SNS/ReverseTraceroute/atlas"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

// Config is the config for the atlas
type Config struct {
	Db dataaccess.DbConfig
}

func init() {
	config.SetEnvPrefix("ATLAS")
	config.AddConfigPath("./atlas.config")
}

func main() {
	go sigHandle()
	conf := Config{}
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	da, err := dataaccess.New(conf.Db)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	svc := atlas.NewAtlasService(da)
	serv := grpc.NewServer()
	pb.RegisterAtlasServer(serv, atlas.GRPCServ{AtlasService: svc})
	ln, err := net.Listen("tcp", "0.0.0.0:55000")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	defer ln.Close()
	err = serv.Serve(ln)
}

func sigHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGSTOP)
	for _ = range c {
		os.Exit(1)
	}
}
