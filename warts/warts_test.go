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
package warts_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/warts"
)

func TestParsePing(t *testing.T) {

	content, err := ioutil.ReadFile("../doc/test_warts.warts")
	if err != nil {
		t.Fatal("ParsePing could not read file")
	}
	_, err = warts.Parse(content)
	if err != nil {
		t.Fatalf("ParsePing failed: %v", err)
	}
}

func TestParsePingTSPreSpec(t *testing.T) {
	content, err := ioutil.ReadFile("../doc/ts_prespec.warts")
	if err != nil {
		t.Fatal("ParsePing could not read file")
	}
	_, err = warts.Parse(content)
	if err != nil {
		t.Fatalf("ParsePing failed: %v", err)
	}
}

func TestParsePingRR(t *testing.T) {
	content, err := ioutil.ReadFile("../doc/rr_test.warts")
	if err != nil {
		t.Fatal("ParsePing could not read file")
	}
	_, err = warts.Parse(content)
	if err != nil {
		t.Fatalf("ParsePing failed: %v", err)
	}
}

func TestTrace(t *testing.T) {
	content, err := ioutil.ReadFile("../doc/trace_test.warts")
	if err != nil {
		t.Fatal("TestTrace could not read file")
	}
	_, err = warts.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
}

var result []interface{}

func BenchmarkParse(b *testing.B) {
	content, err := ioutil.ReadFile("../doc/rr_test.warts")
	if err != nil {
		b.Fatal("ParsePing could not read file")
	}
	b.ResetTimer()
	var res []interface{}
	for i := 0; i < b.N; i++ {
		res, _ = warts.Parse(content)
		for _, item := range res {
			switch i := item.(type) {
			case warts.Ping:
				p := datamodel.Ping{}
				//p.Src = i.Flags.Src.String()
				//p.Dst = i.Flags.Dst.String()
				p.Version = i.Version
				p.Type = i.Type
				//p.Method = i.Flags.PingMethod.String()
				dmt := &datamodel.Time{}
				dmt.Sec = i.Flags.StartTime.Sec
				dmt.Usec = i.Flags.StartTime.Usec
				p.Start = dmt
				p.PingSent = int32(i.Flags.ProbeCount)
				p.ProbeSize = int32(i.Flags.ProbeSize)
				p.Userid = i.Flags.UserID
				p.Ttl = int32(i.Flags.ProbeTTL)
				p.Wait = int32(i.Flags.ProbeWaitS)
				p.Timeout = int32(i.Flags.ProbeTimeout)
				//p.Flags = i.Flags.PF.Strings()
				replies := make([]*datamodel.PingResponse, i.ReplyCount)
				for i, resp := range i.PingReplies {
					rep := &datamodel.PingResponse{}
					rep.From = resp.Addr.String()
					rep.Seq = int32(resp.ProbeID)
					rep.ReplySize = int32(resp.ReplySize)
					rep.ReplyTtl = int32(resp.ReplyTTL)
					rep.ReplyProto = resp.ReplyProto.String()
					txt := &datamodel.Time{}
					txt.Sec = resp.Tx.Sec
					txt.Usec = resp.Tx.Usec
					rep.Tx = txt
					rxt := &datamodel.Time{}
					rxt.Sec = txt.Sec + resp.RTT.Sec
					rxt.Usec = txt.Usec + resp.RTT.Usec
					rep.Rx = rxt
					rep.ProbeIpid = int32(resp.ProbeIPID)
					rep.ReplyIpid = int32(resp.ReplyIPID)
					rep.IcmpType = int32((resp.ICMP & 0xFF00) >> 8)
					rep.IcmpCode = int32(resp.ICMP & 0x00FF)
					if resp.IsTsOnly() {
						rep.Tsonly = make([]int64, 0)
						for _, ts := range resp.V4TS.TimeStamps {
							rep.Tsonly = append(rep.Tsonly, int64(ts))
						}
					} else if resp.IsTsAndAddr() {
						rep.Tsandaddr = make([]*datamodel.TsAndAddr, 0)
						for i, ts := range resp.V4TS.TimeStamps {
							tsa := &datamodel.TsAndAddr{}
							tsa.Ip = resp.V4TS.Addrs[i].String()
							tsa.Ts = int64(ts)
							rep.Tsandaddr = append(rep.Tsandaddr, tsa)
						}
					}
					rep.RR = resp.V4RR.Strings()
					replies[i] = rep
				}
				p.Responses = replies

				stat := i.GetStats()
				pstats := &datamodel.PingStats{}
				pstats.Loss = float32(stat.Loss)
				pstats.Max = stat.Max
				pstats.Min = stat.Min
				pstats.Avg = stat.Avg
				pstats.Stddev = stat.StdDev
				pstats.Replies = int32(stat.Replies)
				p.Statistics = pstats
			}
		}
	}
	result = res
}

func timetoDMTime(t time.Time) *datamodel.Time {
	ret := datamodel.Time{}
	unixNano := t.UnixNano()
	ret.Sec = unixNano / 1000000000
	ret.Usec = (unixNano % 1000000000) / 1000
	return &ret
}
