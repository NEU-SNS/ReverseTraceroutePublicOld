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
package scamper

/*

#cgo LDFLAGS: -lscamperfile
#include <sys/time.h>
#include <stdint.h>
#include <scamper_addr.h>
#include <scamper_ping.h>
#include <scamper_file.h>
#include "warts.h"
import "C"

type Ping struct {
	Version   string
	Type      string
	Method    string
	Src       string
	Dst       string
	Start     *Time
	PingSent  int32
	ProbeSize int32

	Userid     uint32
	Ttl        int32
	Wait       int32
	Timeout    int32
	Flags      []string
	Responses  []*PingResponse
	Statistics *PingStats
}

type PingResponse struct {
	From       string
	Seq        int32
	ReplySize  int32
	ReplyTtl   int32
	ReplyProto string
	Tx         *Time
	Rx         *Time
	Rtt        float32
	ProbeIpid  int32
	ReplyIpid  int32
	IcmpType   int32
	IcmpCode   int32
	RR         []string
	Tsonly     []int64
	Tsandaddr  []*TsAndAddr
}

type PingStats struct {
	Replies int32
	Loss    float32
	Min     float32
	Max     float32
	Avg     float32
	Stddev  float32
}

type Time struct {
	Sec  int64
	Usec int64
}

type TsAndAddr struct {
	Ip string
	Ts int64
}

func ParseWarts(data []byte) (interface{}, error) {
	var result unsafe.Pointer
	var objType C.uint16_t
	if ret := C.parse_warts(unsafe.Pointer(&data[0]), C.size_t(len(data)), &objType, &result); ret != 0 {
		return nil, fmt.Errorf("Failed to parse warts")
	}
	switch objType {
	case C.SCAMPER_FILE_OBJ_PING:
		fmt.Println("Got Ping")
		ret, err := parsePing(result)
		C.free_ping(result)
		return ret, err
	case C.SCAMPER_FILE_OBJ_TRACE:
		fmt.Println("Got Trace")
	}
	return nil, nil
}

func parsePing(ping *C.scamper_ping_t) (Ping, error) {
	types := []string{
		"icmp-echo",
		"tcp-ack",
		"tcp-ack-sport",
		"udp",
		"udp-dport",
		"icmp-time",
		"tcp-syn",
	}
	flags := []string{
		"v4rr",
		"spoof",
		"payload",
		"tsonly",
		"tsandaddr",
		"icmpsum",
		"dl",
		"8",
	}
	var buf [512]byte
	var p Ping
	p.Version = "0.4"
	p.Type = "ping"
	p.Method = C.GoString(types[ping.probe_method])
	p.Src = ping
	return Ping{}, nil
}
*/
