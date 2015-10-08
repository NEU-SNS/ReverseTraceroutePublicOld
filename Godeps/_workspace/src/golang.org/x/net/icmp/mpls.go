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
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icmp

// A MPLSLabel represents a MPLS label stack entry.
type MPLSLabel struct {
	Label int  // label value
	TC    int  // traffic class; formerly experimental use
	S     bool // bottom of stack
	TTL   int  // time to live
}

const (
	classMPLSLabelStack        = 1
	typeIncomingMPLSLabelStack = 1
)

// A MPLSLabelStack represents a MPLS label stack.
type MPLSLabelStack struct {
	Class  int // extension object class number
	Type   int // extension object sub-type
	Labels []MPLSLabel
}

// Len implements the Len method of Extension interface.
func (ls *MPLSLabelStack) Len(proto int) int {
	return 4 + (4 * len(ls.Labels))
}

// Marshal implements the Marshal method of Extension interface.
func (ls *MPLSLabelStack) Marshal(proto int) ([]byte, error) {
	b := make([]byte, ls.Len(proto))
	if err := ls.marshal(proto, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (ls *MPLSLabelStack) marshal(proto int, b []byte) error {
	l := ls.Len(proto)
	b[0], b[1] = byte(l>>8), byte(l)
	b[2], b[3] = classMPLSLabelStack, typeIncomingMPLSLabelStack
	off := 4
	for _, ll := range ls.Labels {
		b[off], b[off+1], b[off+2] = byte(ll.Label>>12), byte(ll.Label>>4&0xff), byte(ll.Label<<4&0xf0)
		b[off+2] |= byte(ll.TC << 1 & 0x0e)
		if ll.S {
			b[off+2] |= 0x1
		}
		b[off+3] = byte(ll.TTL)
		off += 4
	}
	return nil
}

func parseMPLSLabelStack(b []byte) (Extension, error) {
	ls := &MPLSLabelStack{
		Class: int(b[2]),
		Type:  int(b[3]),
	}
	for b = b[4:]; len(b) >= 4; b = b[4:] {
		ll := MPLSLabel{
			Label: int(b[0])<<12 | int(b[1])<<4 | int(b[2])>>4,
			TC:    int(b[2]&0x0e) >> 1,
			TTL:   int(b[3]),
		}
		if b[2]&0x1 != 0 {
			ll.S = true
		}
		ls.Labels = append(ls.Labels, ll)
	}
	return ls, nil
}
