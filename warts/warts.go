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
package warts

import (
	"bytes"
	"fmt"
	"io"
)

type WartsT uint32

const (
	ListT              WartsT = 0x01
	CycleStartT               = 0x02
	CycleDefT                 = 0x03
	CycleStopT                = 0x04
	AddressT                  = 0x05
	TracerouteT               = 0x06
	PingT                     = 0x07
	MDATracerouteT            = 0x08
	AliasResolutionT          = 0x09
	NeighborDiscoveryT        = 0x0a
	TBitT                     = 0x0b
	StingT                    = 0x0c
	SniffT                    = 0x0d
	dummy                     = 0x00
)

func parseNext(f io.Reader) (interface{}, error) {
	head, err := readHeader(f)
	switch head.Type {
	case ListT:
		return readList(f)
	case CycleDefT, CycleStartT:
		return readCycle(f)
	case CycleStopT:
		return readCycleStop(f)
	case AddressT:
	case TracerouteT:
		return readTraceroute(f)
	case PingT:
		return readPing(f)
	case MDATracerouteT:
	case AliasResolutionT:
	case NeighborDiscoveryT:
	case TBitT:
	case StingT:
	case SniffT:
	}
	if err != nil {
		return nil, err
	}
	return head, nil
}

func Parse(data []byte, objs []WartsT) ([]interface{}, error) {
	types := make(map[WartsT]bool)
	for _, obj := range objs {
		types[obj] = true
	}
	ret := make([]interface{}, 0)
	buf := bytes.NewBuffer(data)
	for {
		obj, err := parseNext(buf)
		if err == io.EOF {
			return ret, nil
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to parse warts: %v, %v", err, ret)
		}
		if types[getWartsT(obj)] {
			ret = append(ret, obj)
		}
	}
}

func getWartsT(obj interface{}) WartsT {
	switch obj.(type) {
	case Ping:
		return PingT
	case CycleStart:
		return CycleStartT
	case CycleStop:
		return CycleStopT
	case Traceroute:
		return TracerouteT
	case List:
		return ListT
	default:
		return dummy
	}
}
