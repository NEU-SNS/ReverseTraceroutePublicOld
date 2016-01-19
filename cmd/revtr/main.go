package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
)

func main() {
	flag.Parse()
	var dc dataaccess.DbConfig
	var conf dataaccess.Config
	conf.Host = "localhost"
	conf.Db = "revtr"
	conf.Password = "password"
	conf.Port = "3306"
	conf.User = "revtr"
	dc.ReadConfigs = append(dc.ReadConfigs, conf)
	dc.WriteConfigs = append(dc.WriteConfigs, conf)
	da, err := dataaccess.New(dc)
	if err != nil {
		panic(err)
	}
	_, srvs, err := net.LookupSRV("controller", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	connstr := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	cc, err := grpc.Dial(connstr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	cli := client.New(context.Background(), cc)
	sr := revtr.NewV1Revtr(da)
	h := revtr.NewHome(da, cli)
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("webroot/style"))))
	http.HandleFunc(sr.Route, sr.Handle)
	http.HandleFunc(h.Route, h.Home)
	http.HandleFunc("/runrevtr", func(rw http.ResponseWriter, req *http.Request) {
		log.Debug(req.URL.Query())
		rw.Write([]byte("Hello"))
	})
	for {
		log.Error(http.ListenAndServe(":8080", nil))
	}
}
