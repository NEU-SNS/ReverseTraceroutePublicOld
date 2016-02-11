// Package main is a dummy for testing
package main

import (
	"flag"
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/client"
)

func main() {
	flag.Parse()
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "vpservice.revtr.ccs.neu.edu")
	connvp := fmt.Sprintf("%s:%d", "fring.ccs.neu.edu", 45000)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	cl := client.New(ctx, c3)
	res, err := cl.GetVPs()
	if err != nil {
		panic(err)
	}
	var spoofers []*datamodel.VantagePoint
	for _, vp := range res.GetVps() {
		if vp.CanSpoof {
			fmt.Println(vp)
			spoofers = append(spoofers, vp)
		}
	}
	onePerSite := make(map[string]*datamodel.VantagePoint)
	for _, sp := range spoofers {
		onePerSite[sp.Site] = sp
	}
}
