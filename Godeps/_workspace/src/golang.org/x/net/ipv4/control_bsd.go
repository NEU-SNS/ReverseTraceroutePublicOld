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

// +build darwin dragonfly freebsd netbsd openbsd

package ipv4

import (
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/net/internal/iana"
)

func marshalDst(b []byte, cm *ControlMessage) []byte {
	m := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	m.Level = iana.ProtocolIP
	m.Type = sysIP_RECVDSTADDR
	m.SetLen(syscall.CmsgLen(net.IPv4len))
	return b[syscall.CmsgSpace(net.IPv4len):]
}

func parseDst(cm *ControlMessage, b []byte) {
	cm.Dst = b[:net.IPv4len]
}

func marshalInterface(b []byte, cm *ControlMessage) []byte {
	m := (*syscall.Cmsghdr)(unsafe.Pointer(&b[0]))
	m.Level = iana.ProtocolIP
	m.Type = sysIP_RECVIF
	m.SetLen(syscall.CmsgLen(syscall.SizeofSockaddrDatalink))
	return b[syscall.CmsgSpace(syscall.SizeofSockaddrDatalink):]
}

func parseInterface(cm *ControlMessage, b []byte) {
	sadl := (*syscall.SockaddrDatalink)(unsafe.Pointer(&b[0]))
	cm.IfIndex = int(sadl.Index)
}
