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

// +build !plan9,!solaris,!windows

package ipv4

import (
	"net"
	"syscall"
)

// ReadFrom reads a payload of the received IPv4 datagram, from the
// endpoint c, copying the payload into b.  It returns the number of
// bytes copied into b, the control message cm and the source address
// src of the received datagram.
func (c *payloadHandler) ReadFrom(b []byte) (n int, cm *ControlMessage, src net.Addr, err error) {
	if !c.ok() {
		return 0, nil, nil, syscall.EINVAL
	}
	oob := newControlMessage(&c.rawOpt)
	var oobn int
	switch c := c.PacketConn.(type) {
	case *net.UDPConn:
		if n, oobn, _, src, err = c.ReadMsgUDP(b, oob); err != nil {
			return 0, nil, nil, err
		}
	case *net.IPConn:
		if sockOpts[ssoStripHeader].name > 0 {
			if n, oobn, _, src, err = c.ReadMsgIP(b, oob); err != nil {
				return 0, nil, nil, err
			}
		} else {
			nb := make([]byte, maxHeaderLen+len(b))
			if n, oobn, _, src, err = c.ReadMsgIP(nb, oob); err != nil {
				return 0, nil, nil, err
			}
			hdrlen := int(nb[0]&0x0f) << 2
			copy(b, nb[hdrlen:])
			n -= hdrlen
		}
	default:
		return 0, nil, nil, errInvalidConnType
	}
	if cm, err = parseControlMessage(oob[:oobn]); err != nil {
		return 0, nil, nil, err
	}
	if cm != nil {
		cm.Src = netAddrToIP4(src)
	}
	return
}

// WriteTo writes a payload of the IPv4 datagram, to the destination
// address dst through the endpoint c, copying the payload from b.  It
// returns the number of bytes written.  The control message cm allows
// the datagram path and the outgoing interface to be specified.
// Currently only Darwin and Linux support this.  The cm may be nil if
// control of the outgoing datagram is not required.
func (c *payloadHandler) WriteTo(b []byte, cm *ControlMessage, dst net.Addr) (n int, err error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	oob := marshalControlMessage(cm)
	if dst == nil {
		return 0, errMissingAddress
	}
	switch c := c.PacketConn.(type) {
	case *net.UDPConn:
		n, _, err = c.WriteMsgUDP(b, oob, dst.(*net.UDPAddr))
	case *net.IPConn:
		n, _, err = c.WriteMsgIP(b, oob, dst.(*net.IPAddr))
	default:
		return 0, errInvalidConnType
	}
	if err != nil {
		return 0, err
	}
	return
}
