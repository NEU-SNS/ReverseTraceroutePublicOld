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
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var source, dest string

func init() {
	flag.StringVar(&source, "src", "", "Src")
	flag.StringVar(&dest, "dst", "", "Dst")
}

func main() {
	flag.Parse()
	if source == "" || dest == "" {
		os.Exit(1)
	}
	s, err := os.Open(source)
	if err != nil {
		panic(err)
	}
	var sources, dests []uint32
	scan := bufio.NewScanner(s)
	for scan.Scan() {
		addr := scan.Text()
		if addr == "" {
			continue
		}
		b, err := strconv.ParseUint(addr, 10, 32)
		if err != nil {
			panic(err)
		}
		sources = append(sources, uint32(b))
	}
	err = scan.Err()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(sources), "sources")
	d, err := os.Open(dest)
	if err != nil {
		panic(err)
	}
	scan = bufio.NewScanner(d)
	for scan.Scan() {
		addr := scan.Text()
		if addr == "" {
			continue
		}
		ip, err := util.IPStringToInt32(addr)
		if err != nil {
			panic(err)
		}
		dests = append(dests, ip)
	}
	if err = scan.Err(); err != nil {
		panic(err)
	}
	fmt.Println(len(dests), "dests")
	opts := make([]grpc.DialOption, 1)
	opts[0] = grpc.WithInsecure()
	conn, err := grpc.Dial("rhansen2.revtr.ccs.neu.edu:4380", opts...)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	cl := plcontrollerapi.NewPLControllerClient(conn)
	var ta dm.TracerouteArg
	var tms []*dm.TracerouteMeasurement
	for _, src := range sources {
		for _, dst := range dests {
			tm := new(dm.TracerouteMeasurement)
			tm.Src = src
			tm.Dst = dst
			tm.Timeout = 60 * 10
			tms = append(tms, tm)
		}
	}
	fmt.Println("Running:", len(tms), "traceroutes")
	ta.Traceroutes = tms
	st, err := cl.Traceroute(context.Background())
	if err != nil {
		panic(err)
	}
	st.Send(&ta)
	err = st.CloseSend()
	if err != nil {
		panic(err)
	}
	for {
		tr, err := st.Recv()
		if err == io.EOF {
			fmt.Println("EOF")
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(tr)
	}
}
