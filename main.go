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

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb"
)

func main() {
	flag.Parse()
	vpcreds, err := credentials.NewClientTLSFromFile("/home/rhansen2/sslkey/root.crt", "plcontroller.revtr.ccs.neu.edu")
	connvp := fmt.Sprintf("%s:%d", "walter.ccs.neu.edu", 4380)
	c3, err := grpc.Dial(connvp, grpc.WithTransportCredentials(vpcreds))
	if err != nil {
		panic(err)
	}
	defer c3.Close()
	plc := pb.NewPLControllerClient(c3)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	pm := datamodel.PingMeasurement{
		Src:   2159111452,
		Dst:   69463798,
		Count: "1",
	}
	st, err := plc.Ping(ctx)
	if err != nil {
		panic(err)
	}
	err = st.Send(&datamodel.PingArg{
		Pings: []*datamodel.PingMeasurement{
			&pm,
		},
	})
	if err != nil {
		panic(err)
	}
	err = st.CloseSend()
	if err != nil {
		panic(err)
	}
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(p)
	}
}
