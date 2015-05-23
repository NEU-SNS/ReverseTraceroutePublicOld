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
package main

import (
	"flag"
	"fmt"
	c "github.com/NEU-SNS/ReverseTraceroute/lib/controllerapi"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	conn, err := grpc.Dial("127.0.0.1:35000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	cl := c.NewControllerClient(conn)
	args := dm.PingArg{Service: dm.ServiceT_PLANET_LAB, Dst: "129.10.113.204",
		Host: "127.0.0.1", RR: false}
	ret, err := cl.Ping(con.Background(), &args)
	if err != nil {
		fmt.Printf("Ping failed with err: %v\n", err)
	}
	fmt.Printf("Response took: %s\n", time.Duration(ret.GetRet().Dur))
	a := dm.TracerouteArg{Service: dm.ServiceT_PLANET_LAB, Dst: "8.8.8.8",
		Host: "127.0.0.1"}
	r, err := cl.Traceroute(con.Background(), &a)
	if err != nil {
		fmt.Printf("Traceroute failed with err: %v\n", err)
	}
	fmt.Printf("Response took: %s\n", time.Duration(r.GetRet().Dur))
	arg := dm.StatsArg{Service: dm.ServiceT_PLANET_LAB}
	rr, err := cl.Stats(con.Background(), &arg)
	if err != nil {
		fmt.Printf("Stats failed with err: %v\n", err)
	}
	fmt.Printf("Got back: %v\n", rr)
}
