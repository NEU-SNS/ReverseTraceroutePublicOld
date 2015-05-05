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

type PingArg struct {
	ServiceArg
	Dst   string
	Host  string
	Spoof bool
	RR    bool
	TS    bool
	SAddr string
}

type PingReturn struct {
	ReturnT
	Ping Ping
}

type PingStats struct {
	Replies int     `json:"replies"`
	Loss    int     `json:"loss"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Avg     float64 `json:"avg"`
	Stddev  float64 `json:"stddev"`
}

type PingResponse struct {
	From       string  `json:"from"`
	Seq        int     `json:"seq"`
	ReplySize  int     `json:"reply_size"`
	ReplyTtl   int     `json:"reply_ttl"`
	ReplyProto string  `json:"reply_proto"`
	Tx         Time    `json:"tx"`
	Rx         Time    `json:"rx"`
	Rtt        float64 `json:"rtt"`
	ProbeIpId  int     `json:"probe_ipid"`
	ReplyIpId  int     `json:"reply_ipid"`
	IcmpType   int     `json:"icmp_type"`
	IcmpCode   int     `json:"icmp_code"`
}

type Ping struct {
	Version    string         `json:"version"`
	Type       string         `json:"type"`
	Method     string         `json:"method"`
	Src        string         `json:"src"`
	Dst        string         `json:"dst"`
	Start      Time           `json:"start"`
	PingSent   int            `json:"ping_sent"`
	ProbeSize  int            `json:"probe_size"`
	UserId     int            `json:"userid"`
	Ttl        int            `json:"ttl"`
	Wait       int            `json:"wait"`
	Timeout    int            `json:"timeout"`
	Responses  []PingResponse `json:"responses"`
	Statistics PingStats      `json:"statistics"`
}

func createPing() interface{} {
	return new(Ping)
}
