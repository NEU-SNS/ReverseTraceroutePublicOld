// Package main is a dummy for testing
package main

import (
	"fmt"
	"net"

	"golang.org/x/net/context"

	vpserv "github.com/NEU-SNS/ReverseTraceroute/vpservice/client"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	_, srvs, err := net.LookupSRV("vpservice", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", srvs[0].Target)
	connvp := fmt.Sprintf("%s:%d", srvs[0].Target, srvs[0].Port)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	vcl := vpserv.New(context.Background(), c3)
	defer c3.Close()
	vps, err := vcl.GetVPs()
	if err != nil {
		panic(err)
	}
	for _, vp := range vps.GetVps() {
		if vp.CanSpoof {
			fmt.Println(vp.Hostname)
		}
	}
}
