/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package main is a dummy for testing
package main

import (
	"flag"
	"fmt"
	"io"
	"runtime"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	opts := make([]grpc.DialOption, 1)
	opts[0] = grpc.WithInsecure()
	conn, err := grpc.Dial("rhansen2.revtr.ccs.neu.edu:4380", opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	cl := plc.NewPLControllerClient(conn)
	vps, err := cl.GetVPs(ctx.Background(), &dm.VPRequest{})
	if err != nil {
		panic(err)
	}

	vplist := make([]*dm.VantagePoint, 0)
	for {
		vpp, err := vps.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		vplist = append(vplist, vpp.GetVps()...)
	}
	pingreq := &dm.PingArg{
		Pings: make([]*dm.PingMeasurement, 0),
	}
	fmt.Println("Num of vps: ", len(vplist))
	dst := new(string)
	for i, vp := range vplist {
		if i == 0 {
			ip, err := util.Int32ToIPString(vp.Ip)
			if err != nil {
				panic(err)
			}
			dst = &ip
			continue
		}
		if vp.Controller != 0 {
			src, err := util.Int32ToIPString(vp.Ip)
			if err != nil {
				panic(err)
			}
			pingreq.Pings = append(pingreq.Pings, &dm.PingMeasurement{
				Src: src,
				Dst: *dst,
				RR:  true,
			})
		}
	}
	//fmt.Println(pingreq)
	fmt.Println("Num of requests: ", len(pingreq.Pings))
	fmt.Println("Starting: ", time.Now())
	st, err := cl.Ping(ctx.Background(), pingreq)
	if err != nil {
		panic(err)
	}
	ps := make([]*dm.Ping, 0)
	for {
		pr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		//fmt.Println(pr)
		ps = append(ps, pr)
	}
	for _, p := range ps {
		resp := p.GetResponses()
		if resp == nil || len(resp) == 0 {
			continue
		}
		for _, re := range resp {
			if len(re.RR) > 0 {
				fmt.Println(p.Src, "RR:", re.RR)
				for i, r := range re.RR {
					if r == p.Src {
						fmt.Println(p.Src, "includes src in RR at addr:", i)
					}
				}
			}
		}
	}
	fmt.Println("Done: ", time.Now())
}
