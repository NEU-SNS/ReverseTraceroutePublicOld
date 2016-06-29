package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/api"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	"github.com/NEU-SNS/ReverseTraceroute/config"
	cclient "github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/httputils"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	vpsclient "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
)

// Config is the config for the atlas
type Config struct {
	DB       repo.Configs
	RootCA   string `flag:"root-ca"`
	CertFile string `flag:"cert-file"`
	KeyFile  string `flag:"key-file"`
}

func init() {
	config.SetEnvPrefix("ATLAS")
	config.AddConfigPath("./atlas.config")
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
	conf := Config{}
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
	r, err := repo.NewRepo(repoOpts...)
	if err != nil {
		log.Fatal(err)
	}
	vps, err := makeVPS(conf.RootCA)
	if err != nil {
		log.Fatal(err)
	}
	cc, err := makeClient(conf.RootCA)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/metrics", prometheus.Handler())
	go func() {
		for {
			log.Error(http.ListenAndServe(":8080", nil))
		}
	}()
	cache, err := lru.New(1000000)
	if err != nil {
		panic("Could not create cache " + err.Error())
	}
	serv := server.NewServer(server.WithVPS(vps),
		server.WithTRS(r),
		server.WithClient(cc),
		server.WithCache(cache))
	ln, err := net.Listen("tcp", ":55000")
	if err != nil {
		log.Fatal(err)
	}
	defer logError(ln.Close)
	tlsc, err := httputil.TLSConfig(conf.CertFile, conf.KeyFile)
	s := api.CreateServer(serv, tlsc)
	err = s.Serve(ln)
	if err != nil {
		log.Fatal(err)
	}
}

func makeVPS(rootCA string) (vpsclient.VPSource, error) {
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return nil, err
	}
	creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		return nil, err
	}
	conn := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c, err := grpc.Dial(conn, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return vpsclient.New(context.Background(), c), nil
}

func makeClient(rootCA string) (cclient.Client, error) {
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		return nil, err
	}
	creds, err := credentials.NewClientTLSFromFile(rootCA, srvs[0].Target)
	if err != nil {
		return nil, err
	}
	conn := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c, err := grpc.Dial(conn, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return cclient.New(context.Background(), c), nil
}
