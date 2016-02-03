// Package main is a dummy for testing
package main

import (
	"flag"
	"fmt"
	"io"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

func main() {
	flag.Parse()
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "atlas.revtr.ccs.neu.edu")
	connvp := fmt.Sprintf("%s:%d", "rhansen2.revtr.ccs.neu.edu", 55000)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	cl := client.New(context.Background(), c3)
	s, err := cl.GetIntersectingPath(context.Background())
	if err != nil {
		panic(err)
	}
	s.Send(&datamodel.IntersectionRequest{
		Address:    2587245126,
		Dest:       644601874,
		Staleness:  60,
		UseAliases: true,
	})
	s.CloseSend()
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
