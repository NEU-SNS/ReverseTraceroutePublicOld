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
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

func main() {
	flag.Parse()
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "controller.revtr.ccs.neu.edu")
	connvp := fmt.Sprintf("%s:%d", "fring.ccs.neu.edu", 4382)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	cl := client.New(context.Background(), c3)
	saddr, _ := util.Int32ToIPString(2164947137)
	s, err := cl.Ping(ctx, &datamodel.PingArg{Pings: []*datamodel.PingMeasurement{
		&datamodel.PingMeasurement{
			Src:   2170636814,
			SAddr: saddr,
			Dst:   2162100337,
			Spoof: true,
			Count: "1",
		},
	}})
	if err != nil {
		panic(err)
	}
	for {
		r, err := s.Recv()
		if err == io.EOF {
			log.Info("done")
			break
		}
		if err != nil {
			log.Error(err)
			return
		}
		log.Info(r)
	}
}
