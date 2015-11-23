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

import "flag"

func main() {
	flag.Parse()
	/*
		opts := make([]grpc.DialOption, 1)
		opts[0] = grpc.WithInsecure()
		conn, err := grpc.Dial("rhansen2.revtr.ccs.neu.edu:4382", opts...)
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		c := controllerapi.NewControllerClient(conn)
		ct := context.Background()
		ctx, cancel := context.WithTimeout(ct, time.Second*70)
		defer cancel()
		vpr, err := c.GetVPs(ctx, &dm.VPRequest{})
		if err != nil {
			panic(err)
		}
		vps := vpr.GetVps()
		var pa dm.PingArg
		var pings []*dm.PingMeasurement
		var dst uint32 = 2164945295
		for _, vp := range vps {
			pings = append(pings, &dm.PingMeasurement{
				Src:     vp.Ip,
				Dst:     dst,
				Timeout: 60,
				Count:   "1",
				Spoof:   true,
				SAddr:   "204.8.155.227",
				RR:      true,
				//CheckCache: true,
				//CheckDb: true,
			})

		}
		pa.Pings = pings
		fmt.Println("Number of requests:", len(pings))
		start := time.Now()
		fmt.Println("Starting:", start)
		st, err := c.Ping(context.Background(), &pa)
		var ps []*dm.Ping
		if err != nil {
			panic(err)
		}
		for {
			pr, err := st.Recv()
			if err == io.EOF {
				log.Info("EOF")
				break
			}
			if err != nil {
				panic(err)
			}
			ps = append(ps, pr)
		}
		end := time.Now()
		fmt.Println("Done:", end, "Took:", time.Since(start), "Got: ", len(ps), "spoofs")
		fmt.Print(ps)
	*/
	/*
		opts := make([]grpc.DialOption, 1)
		opts[0] = grpc.WithInsecure()
		conn, err := grpc.Dial("rhansen2.revtr.ccs.neu.edu:4382", opts...)
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		cl := controllerapi.NewControllerClient(conn)
		var pa dm.PingArg
		var pings []*dm.PingMeasurement
		var dst uint32 = 2164945295
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			addr := scanner.Text()
			ip, err := util.IPStringToInt32(addr)
			if err != nil {
				panic(err)
			}
			pings = append(pings, &dm.PingMeasurement{
				Src:     ip,
				Dst:     dst,
				Timeout: 60,
				Count:   "1",
				Spoof:   true,
				SAddr:   "204.8.155.227",
				//CheckCache: true,
				//CheckDb: true,
			})
		}
		pa.Pings = pings
		s, err := cl.Ping(context.Background(), &pa)
		if err != nil {
			panic(err)
		}
		var ps []*dm.Ping
		for {
			pr, err := s.Recv()
			if err == io.EOF {
				log.Info("EOF")
				break
			}
			if err != nil {
				panic(err)
			}
			ps = append(ps, pr)
		}
		for _, p := range ps {
			fmt.Println(p.SpoofedFrom)
		}
	*/
}
