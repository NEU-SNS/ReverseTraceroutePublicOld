// Package main is a dummy for testing
package main

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	creds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "controller.revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", "controller.revtr.ccs.neu.edu", 4382), grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	cl := client.New(context.Background(), cc)
	vps, err := cl.GetVps(&datamodel.VPRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Println(vps)
}
