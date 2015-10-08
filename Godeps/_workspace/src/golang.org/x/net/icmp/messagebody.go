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
// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icmp

// A MessageBody represents an ICMP message body.
type MessageBody interface {
	// Len returns the length of ICMP message body.
	// Proto must be either the ICMPv4 or ICMPv6 protocol number.
	Len(proto int) int

	// Marshal returns the binary enconding of ICMP message body.
	// Proto must be either the ICMPv4 or ICMPv6 protocol number.
	Marshal(proto int) ([]byte, error)
}

// A DefaultMessageBody represents the default message body.
type DefaultMessageBody struct {
	Data []byte // data
}

// Len implements the Len method of MessageBody interface.
func (p *DefaultMessageBody) Len(proto int) int {
	if p == nil {
		return 0
	}
	return len(p.Data)
}

// Marshal implements the Marshal method of MessageBody interface.
func (p *DefaultMessageBody) Marshal(proto int) ([]byte, error) {
	return p.Data, nil
}

// parseDefaultMessageBody parses b as an ICMP message body.
func parseDefaultMessageBody(proto int, b []byte) (MessageBody, error) {
	p := &DefaultMessageBody{Data: make([]byte, len(b))}
	copy(p.Data, b)
	return p, nil
}
