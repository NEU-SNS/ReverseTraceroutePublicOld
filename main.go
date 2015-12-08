// Package main is a dummy for testing
package main

import (
	"fmt"
	"io"
	"net"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/client"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

func main() {
	_, servs, err := net.LookupSRV("atlas", "tcp", "revtr.ccs.neu.edu")
	if err != nil {
		panic(err)
	}
	srv := servs[0]
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", srv.Target, srv.Port), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	cl := client.New(context.Background(), cc)
	st, err := cl.GetIntersectingPath()
	if err != nil {
		panic(err)
	}
	req := &datamodel.IntersectionRequest{
		Address:    71593665,
		Dest:       68101007,
		UseAliases: true,
	}
	if err := st.Send(req); err != nil {
		panic(err)
	}
	st.CloseSend()
	fmt.Println("Close Send")
	for {
		res, err := st.Recv()
		if err == io.EOF {
			fmt.Println("done")
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(res)
	}
}
