// Package main is a dummy for testing
package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/controller/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

func main() {
	flag.Parse()
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "controller.revtr.ccs.neu.edu")
	connvp := fmt.Sprintf("%s:%d", "walter.ccs.neu.edu", 4382)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	cc := client.New(context.Background(), c3)
	pm := datamodel.PingMeasurement{
		Src:        2915894044,
		Dst:        1023933481,
		SAddr:      "61.7.252.28",
		Spoof:      true,
		RR:         true,
		CheckCache: true,
		CheckDb:    true,
		Count:      "1",
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	c, err := cc.Ping(ctx, &datamodel.PingArg{
		Pings: []*datamodel.PingMeasurement{&pm},
	})
	if err != nil {
		panic(err)
	}
	c.CloseSend()
	for {
		p, err := c.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(p)
	}
}
