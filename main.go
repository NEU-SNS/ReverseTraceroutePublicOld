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

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	ctx "golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	conn, err := grpc.Dial("129.10.113.205:45000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	cl := plc.NewPLControllerClient(conn)
	pa := &dm.PingArg{Pings: []*dm.PingMeasurement{&dm.PingMeasurement{Src: "129.10.113.205", Dst: "8.8.8.8"}, &dm.PingMeasurement{Src: "129.10.113.205", Dst: "8.8.4.4"}}}
	stream, err := cl.Ping(ctx.Background(), pa)
	if err != nil {
		panic(err)
	}
	ta := &dm.TracerouteArg{Traceroutes: []*dm.TracerouteMeasurement{&dm.TracerouteMeasurement{Src: "129.10.113.205", Dst: "8.8.8.8"}, &dm.TracerouteMeasurement{Src: "129.10.113.205", Dst: "8.8.4.4"}}}
	st, err := cl.Traceroute(ctx.Background(), ta)
	if err != nil {
		panic(err)
	}
	for {
		ping, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(ping)
	}

	for {
		trace, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(trace)
	}
	fmt.Println("Got all measurements")
}
