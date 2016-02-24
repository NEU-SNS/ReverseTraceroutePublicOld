package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
	"github.com/prometheus/client_golang/prometheus"
)

var conf = revtr.NewConfig()

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
}

func main() {
	err := config.Parse(flag.CommandLine, &conf)
	if err != nil {
		log.Error(err)
	}
	da, err := dataaccess.New(conf.Db)
	if err != nil {
		panic(err)
	}
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	ccreds, err := credentials.NewClientTLSFromFile(*conf.RootCA, srvs[0].Target)
	if err != nil {
		panic(err)
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	cc, err := grpc.Dial(connstr, grpc.WithTransportCredentials(ccreds))
	if err != nil {
		panic(err)
	}

	_, srvs, err = net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	vpcreds, err := credentials.NewClientTLSFromFile(*conf.RootCA, srvs[0].Target)
	if err != nil {
		panic(err)
	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	vps := vpservice.New(context.Background(), c3)
	cli := client.New(context.Background(), cc)
	sr := revtr.NewV1Revtr(da, vps, *conf.RootCA)
	h := revtr.NewHome(da, cli, vps)
	srcs := revtr.NewV1Sources(da, *conf.RootCA, vps)
	runrtr := revtr.NewRunRevtr(da, *conf.RootCA)
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("webroot/style"))))
	http.HandleFunc(sr.Route, sr.Handle)
	http.HandleFunc(h.Route, h.Home)
	http.HandleFunc(runrtr.Route, runrtr.RunRevtr)
	http.HandleFunc(srcs.Route, srcs.Handle)
	http.HandleFunc("/ws", runrtr.WS)
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
		log.Error(http.ListenAndServeTLS(":8080", *conf.CertFile, *conf.KeyFile, nil))
	}
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
