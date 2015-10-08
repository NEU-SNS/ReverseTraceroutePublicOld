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

package icmp

import "golang.org/x/net/internal/iana"

// A ParamProb represents an ICMP parameter problem message body.
type ParamProb struct {
	Pointer    uintptr     // offset within the data where the error was detected
	Data       []byte      // data, known as original datagram field
	Extensions []Extension // extensions
}

// Len implements the Len method of MessageBody interface.
func (p *ParamProb) Len(proto int) int {
	if p == nil {
		return 0
	}
	l, _ := multipartMessageBodyDataLen(proto, p.Data, p.Extensions)
	return l
}

// Marshal implements the Marshal method of MessageBody interface.
func (p *ParamProb) Marshal(proto int) ([]byte, error) {
	if proto == iana.ProtocolIPv6ICMP {
		b := make([]byte, 4+p.Len(proto))
		b[0], b[1], b[2], b[3] = byte(p.Pointer>>24), byte(p.Pointer>>16), byte(p.Pointer>>8), byte(p.Pointer)
		copy(b[4:], p.Data)
		return b, nil
	}
	b, err := marshalMultipartMessageBody(proto, p.Data, p.Extensions)
	if err != nil {
		return nil, err
	}
	b[0] = byte(p.Pointer)
	return b, nil
}

// parseParamProb parses b as an ICMP parameter problem message body.
func parseParamProb(proto int, b []byte) (MessageBody, error) {
	if len(b) < 4 {
		return nil, errMessageTooShort
	}
	p := &ParamProb{}
	if proto == iana.ProtocolIPv6ICMP {
		p.Pointer = uintptr(b[0])<<24 | uintptr(b[1])<<16 | uintptr(b[2])<<8 | uintptr(b[3])
		p.Data = make([]byte, len(b)-4)
		copy(p.Data, b[4:])
		return p, nil
	}
	p.Pointer = uintptr(b[0])
	var err error
	p.Data, p.Extensions, err = parseMultipartMessageBody(proto, b)
	if err != nil {
		return nil, err
	}
	return p, nil
}
