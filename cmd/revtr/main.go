package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/config"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
)

var conf = revtr.NewConfig()

func init() {
	config.SetEnvPrefix("REVTR")
	config.AddConfigPath("./revtr.config")
}

func main() {
	config.Parse(flag.CommandLine, &conf)
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
	sr := revtr.NewV1Revtr(da, *conf.RootCA)
	h := revtr.NewHome(da, cli, vps)
	runrtr := revtr.NewRunRevtr(da, *conf.RootCA)
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("webroot/style"))))
	http.HandleFunc(sr.Route, sr.Handle)
	http.HandleFunc(h.Route, h.Home)
	http.HandleFunc(runrtr.Route, runrtr.RunRevtr)
	http.HandleFunc("/ws", runrtr.WS)
	for {
		log.Error(http.ListenAndServe(":8080", nil))
	}
}
