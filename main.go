// Package main is a dummy for testing
package main

import (
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"

	"github.com/NEU-SNS/ReverseTraceroute/plcontroller/pb"
	"google.golang.org/grpc"
)

func main() {
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", "plcontroller.revtr.ccs.neu.edu", 4380), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	cl := pb.NewPLControllerClient(cc)
	var pings []*datamodel.PingMeasurement
	ipToHost := make(map[uint32]*datamodel.VantagePoint)
	vps, err := cl.GetVPs(context.Background(), &datamodel.VPRequest{})
	for {
		v, err := vps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		vv := v.GetVps()
		for _, vp := range vv {
			ipToHost[vp.Ip] = vp
			p := &datamodel.PingMeasurement{
				Src:   vp.Ip,
				Dst:   2164945295,
				Count: "1",
			}
			pings = append(pings, p)
		}
	}
	pm := datamodel.PingArg{
		Pings: pings,
	}
	st, err := cl.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	st.Send(&pm)
	st.CloseSend()
	var res []*datamodel.Ping
	var errs []*datamodel.Ping
	for {
		p, err := st.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			panic(err)
		}
		if p.Error == "" {
			res = append(res, p)
		} else {
			errs = append(errs, p)
		}
	}
	for _, print := range res {
		if _, ok := ipToHost[print.Src]; !ok {
			continue
		}
		fmt.Println(ipToHost[print.Src].Hostname, " ", ipToHost[print.Src].Site)
	}
}
