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
package datamodel

import (
	"github.com/NEU-SNS/ReverseTraceroute/warts"
	"github.com/gogo/protobuf/proto"
)

func createPing() interface{} {
	return new(Ping)
}
func (p *Ping) CUnmarshal(data []byte) error {
	return proto.Unmarshal(data, p)
}

/*
func (p *Ping) Marshal() ([]byte, error) {
	return proto.Marshal(p)
}

func (pm *PingMeasurement) Marshal() ([]byte, error) {
	return proto.Marshal(pm)
}
*/
func (pm *PingMeasurement) Key() string {
	return ""
}

func (p *Ping) Key() string {
	return ""
}

func ConvertPing(in warts.Ping) Ping {
	p := Ping{}
	p.Src = in.Flags.Src.String()
	p.Dst = in.Flags.Dst.String()
	p.Version = in.Version
	p.Type = in.Type
	p.Method = in.Flags.PingMethod.String()
	dmt := &Time{}
	dmt.Sec = in.Flags.StartTime.Sec
	dmt.Usec = in.Flags.StartTime.Usec
	p.Start = dmt
	p.PingSent = int32(in.Flags.ProbeCount)
	p.ProbeSize = int32(in.Flags.ProbeSize)
	p.Userid = in.Flags.UserID
	p.Ttl = int32(in.Flags.ProbeTTL)
	p.Wait = int32(in.Flags.ProbeWaitS)
	p.Timeout = int32(in.Flags.ProbeTimeout)
	p.Flags = in.Flags.PF.Strings()
	replies := make([]*PingResponse, in.ReplyCount)
	for i, resp := range in.PingReplies {
		rep := &PingResponse{}
		rep.From = resp.Addr.String()
		rep.Seq = int32(resp.ProbeID)
		rep.ReplySize = int32(resp.ReplySize)
		rep.ReplyTtl = int32(resp.ReplyTTL)
		rep.ReplyProto = resp.ReplyProto.String()
		txt := &Time{}
		txt.Sec = resp.Tx.Sec
		txt.Usec = resp.Tx.Usec
		rep.Tx = txt
		rxt := &Time{}
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
			rep.Tsandaddr = make([]*TsAndAddr, 0)
			for i, ts := range resp.V4TS.TimeStamps {
				tsa := &TsAndAddr{}
				tsa.Ip = resp.V4TS.Addrs[i].String()
				tsa.Ts = int64(ts)
				rep.Tsandaddr = append(rep.Tsandaddr, tsa)
			}
		}
		rep.RR = resp.V4RR.Strings()
		replies[i] = rep
	}
	p.Responses = replies
	stat := in.GetStats()
	pstats := &PingStats{}
	pstats.Loss = float32(stat.Loss)
	pstats.Max = stat.Max
	pstats.Min = stat.Min
	pstats.Avg = stat.Avg
	pstats.Stddev = stat.StdDev
	pstats.Replies = int32(stat.Replies)
	p.Statistics = pstats

	return p
}
