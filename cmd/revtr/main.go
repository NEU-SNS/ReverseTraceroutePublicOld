package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/pb"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/server"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/types"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/v1api"
	"github.com/NEU-SNS/ReverseTraceroute/revtr/v2api"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/gengo/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhansen2/ReverseTraceroute/revtr"
)

var (
	conf            = revtr.NewConfig()
	homeTemplate    = template.Must(template.ParseFiles("webroot/templates/home.html"))
	runningTemplate = template.Must(template.ParseFiles("webroot/templates/running.html"))
)

// AppConfig for the app
type AppConfig struct {
	ServerConfig types.Config
	DB           dataaccess.DbConfig
}

func init() {
	config.SetEnvPrefix("REVTR")
	config.AddConfigPath("./revtr.config")
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

func tlsConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}

func main() {
	conf := AppConfig{
		ServerConfig: types.NewConfig(),
	}
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Error(err)
	}
	da, err := dataaccess.New(conf.DB)
	if err != nil {
		panic(err)
	}
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)

	}
	vpcreds, err := credentials.NewClientTLSFromFile(*conf.ServerConfig.RootCA, srvs[0].Target)
	if err != nil {
		panic(err)

	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	vps := vpservice.New(context.Background(), c3)

	tlsConf, err := tlsConfig(*conf.ServerConfig.CertFile, *conf.ServerConfig.KeyFile)
	if err != nil {
		panic(err)
	}
	serv := server.NewRevtrServer(server.WithVPSource(vps),
		server.WithAdjacencySource(da),
		server.WithClusterSource(da),
		server.WithRootCA(*conf.ServerConfig.RootCA),
		server.WithCertFile(*conf.ServerConfig.CertFile),
		server.WithKeyFile(*conf.ServerConfig.KeyFile))
	mux := http.NewServeMux()
	mux.Handle("/styles/", http.StripPrefix("/styles", http.FileServer(http.Dir("webroot/style"))))
	v1api.NewV1Api(serv, mux)
	v2serv := v2api.CreateServer(serv, tlsConf)
	gatewayMux := runtime.NewServeMux()
	selfCreds, err := credentials.NewClientTLSFromFile(*conf.ServerConfig.RootCA, "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(selfCreds)}
	err = pb.RegisterRevtrHandlerFromEndpoint(context.Background(), gatewayMux, ":8080", dialOpts)
	if err != nil {
		panic(err)
	}
	conn, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	s := &http.Server{
		Addr:      ":8080",
		TLSConfig: tlsConf,
		Handler:   directRequest(v2serv, mux),
	}

	metricsServ := http.NewServeMux()
	metricsServ.Handle("/metrics", prometheus.Handler())
	go func() {
		for {
			log.Error(http.ListenAndServe(":45454", metricsServ))
		}
	}()
	go func() {
		for {
			log.Error(http.ListenAndServe(":8181", http.HandlerFunc(redirect)))
		}
	}()
	for {
		log.Error(s.Serve(tls.NewListener(conn, tlsConf)))
	}
}

func directRequest(grpcServer *grpc.Server, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func redirect(w http.ResponseWriter, req *http.Request) {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			log.Error(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		host = req.Host
	}
	http.Redirect(w, req, "https://"+host+":443"+req.RequestURI, http.StatusMovedPermanently)
}
