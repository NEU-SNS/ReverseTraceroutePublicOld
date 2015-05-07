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
	"errors"
	"time"
)

type TracerouteArg struct {
	ServiceArg
	Host         string
	Dst          string
	Confidence   string
	DPort        string
	FirstHop     string
	GapLimit     string
	GapAction    string
	MaxTtl       string
	PathDiscov   bool
	Loops        string
	LoopAction   string
	Payload      string
	Method       string
	Attempts     string
	SendAll      bool
	SPort        string
	SAddr        string
	Tos          string
	TimeExceeded bool
	UserId       string
	Wait         string
	WaitProbe    string
	GssEntry     string
	LssName      string
}

type Traceroute struct {
	Version    string          `json:"version"`
	Type       string          `json:"type"`
	UserId     int             `json:"userid"`
	Method     string          `json:"method"`
	Src        string          `json:"src"`
	Dst        string          `json:"dst"`
	SPort      int             `json:"sport"`
	DPort      int             `json:"dport"`
	StopReason string          `json:"stop_reason"`
	StopData   int             `json:"stop_data"`
	Start      TracerouteTime  `json:"start"`
	HopCount   int             `json:"hop_count"`
	Attempts   int             `json:"attempts"`
	HopLimit   int             `json:"hoplimit"`
	FirstHop   int             `json:"firsthop"`
	Wait       int             `json:"wait"`
	WaitProbe  int             `json:"wait_probe"`
	Tos        int             `json:"tos"`
	ProbeSize  int             `json:"probe_size"`
	Hops       []TracerouteHop `json:"hops"`
}

type TracerouteHop struct {
	Addr      string  `json:"addr"`
	ProbeTtl  int     `json:"probe_ttl"`
	ProbeId   int     `json:"probe_id"`
	ProbeSize int     `json:"probe_size"`
	Rtt       float32 `json:"rtt"`
	ReplyTtl  int     `json:"reply_ttl"`
	ReplyTos  int     `json:"reply_tos"`
	ReplySize int     `json:"reply_size"`
	ReplyIpId int     `json:"reply_ipid"`
	IcmpType  int     `json:"icmp_type"`
	IcmpCode  int     `json:"icmp_code"`
	IcmpQTtl  int     `json:"icmp_q_ttl"`
	IcmpqIpl  int     `json:"icmp_q_ipl"`
	IcmpQTos  int     `json:"icmp_q_tos"`
}

type TTime time.Time

const TRACETIME = "2006-01-_2 15:04:05"

func (t *TTime) UnmarshalJSON(data []byte) (err error) {
	temp, err := time.Parse(`"`+TRACETIME+`"`, string(data))
	*t = TTime(temp)
	return
}

func (t TTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if y := tt.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}
	return []byte(tt.Format(`"` + TRACETIME + `"`)), nil
}

func (t TTime) String() string {
	tt := time.Time(t)
	return tt.String()
}

type TracerouteTime struct {
	Sec   int64 `json:"sec"`
	USec  int64 `json:"usec"`
	Ftime TTime `json:"ftime"`
}

type TracerouteReturn struct {
	ReturnT
	Traceroute Traceroute
}

func createTraceroute() interface{} {
	return new(Traceroute)
}
