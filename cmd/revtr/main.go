package main

import (
	"flag"
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	at "github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/revtr"
	vpservice "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
)

func main() {
	flag.Parse()
	test := revtr.ReverseTracerouteReq{
		Dst: "8.8.8.8",
		Src: "4.71.254.141",
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
	defer cc.Close()
	cli := client.New(context.Background(), cc)
	_, srvs, err = net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	connstrat := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c2, err := grpc.Dial(connstrat, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer c2.Close()
	atl := at.New(context.Background(), c2)
	_, srvs, err = net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	vps := vpservice.New(context.Background(), c3)

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
	rt, re, c, err := revtr.RunReverseTraceroute(test, true, cli, atl, vps, da, "/home/rhansen2/dev/go/src/github.com/NEU-SNS/ReverseTraceroute/revtr/alias_lists.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rt, re, c)
}
