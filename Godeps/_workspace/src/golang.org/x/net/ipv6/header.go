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
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ipv6

import (
	"errors"
	"fmt"
	"net"
)

const (
	Version   = 6  // protocol version
	HeaderLen = 40 // header length
)

// A Header represents an IPv6 base header.
type Header struct {
	Version      int    // protocol version
	TrafficClass int    // traffic class
	FlowLabel    int    // flow label
	PayloadLen   int    // payload length
	NextHeader   int    // next header
	HopLimit     int    // hop limit
	Src          net.IP // source address
	Dst          net.IP // destination address
}

func (h *Header) String() string {
	if h == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ver: %v, tclass: %#x, flowlbl: %#x, payloadlen: %v, nxthdr: %v, hoplim: %v, src: %v, dst: %v", h.Version, h.TrafficClass, h.FlowLabel, h.PayloadLen, h.NextHeader, h.HopLimit, h.Src, h.Dst)
}

// ParseHeader parses b as an IPv6 base header.
func ParseHeader(b []byte) (*Header, error) {
	if len(b) < HeaderLen {
		return nil, errors.New("header too short")
	}
	h := &Header{
		Version:      int(b[0]) >> 4,
		TrafficClass: int(b[0]&0x0f)<<4 | int(b[1])>>4,
		FlowLabel:    int(b[1]&0x0f)<<16 | int(b[2])<<8 | int(b[3]),
		PayloadLen:   int(b[4])<<8 | int(b[5]),
		NextHeader:   int(b[6]),
		HopLimit:     int(b[7]),
	}
	h.Src = make(net.IP, net.IPv6len)
	copy(h.Src, b[8:24])
	h.Dst = make(net.IP, net.IPv6len)
	copy(h.Dst, b[24:40])
	return h, nil
}
