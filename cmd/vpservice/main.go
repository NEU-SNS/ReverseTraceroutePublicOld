package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/httputils"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/api"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/filters"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/repo"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/types"
)

// AppConfig is the config struct for the atlas
type AppConfig struct {
	ServerConfig types.Config
	DB           repo.Configs
}

func init() {
	config.SetEnvPrefix("VPS")
	config.AddConfigPath("./vpservice.config")
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		switch {
		case err != nil:
			return false, false
		case host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "rhansen2.local" || host == "rhansen2.revtr.ccs.neu.edu" || host == "129.10.113.189":
			return true, true
		default:
			return false, false
		}
	}
	grpclog.SetLogger(log.GetLogger())
}

type errorf func() error

func logError(f errorf) {
	if err := f(); err != nil {
		log.Error(err)
	}
}

func main() {
	conf := AppConfig{
		ServerConfig: types.NewConfig(),
	}
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Fatal(err)
	}
	var repoOpts []repo.Option
	for _, c := range conf.DB.WriteConfigs {
		repoOpts = append(repoOpts, repo.WithWriteConfig(c))
	}
	for _, c := range conf.DB.ReadConfigs {
		repoOpts = append(repoOpts, repo.WithReadConfig(c))
	}
	da, err := repo.NewRepo(repoOpts...)
	if err != nil {
		log.Fatal(err)
	}
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		log.Fatal(err)
	}
	ccreds, err := credentials.NewClientTLSFromFile(*conf.ServerConfig.RootCA, srvs[0].Target)
	if err != nil {
		log.Fatal(err)
	}
	connst := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c, err := grpc.Dial(connst, grpc.WithTransportCredentials(ccreds))
	if err != nil {
		log.Fatal(err)
	}
	defer logError(c.Close)
	cl := client.New(context.Background(), c)
	rrf, tsf := makeFilters()
	s, err := server.NewServer(server.WithVPProvider(da),
		server.WithClient(cl),
		server.WithRRFilter(rrf),
		server.WithTSFilter(tsf))

	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", ":45000")
	if err != nil {
		log.Fatal(err)
	}
	defer logError(ln.Close)
	tlsc, err := httputil.TLSConfig(*conf.ServerConfig.CertFile, *conf.ServerConfig.KeyFile)
	if err != nil {
		log.Error(err)
		log.Error(conf)
		if err := ln.Close(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}
	apiServ := api.CreateServer(s, tlsc)
	err = apiServ.Serve(ln)
	if err != nil {
		log.Fatal(err)
	}
}

func makeFilters() (filters.RRFilter, filters.TSFilter) {
	rrf := filters.ComposeRRFilter(filters.MakeRRDistanceFilter(9, 9),
		filters.OnePerSiteRR,
		filters.OrderRRDistanceFilter)
	return rrf, filters.OnePerSiteTS
}
