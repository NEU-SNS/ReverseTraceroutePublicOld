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

// +build darwin freebsd linux

package ipv4

import (
	"net"
	"os"
	"unsafe"

	"golang.org/x/net/internal/iana"
)

var freebsd32o64 bool

func setsockoptGroupReq(fd, name int, ifi *net.Interface, grp net.IP) error {
	var gr sysGroupReq
	if ifi != nil {
		gr.Interface = uint32(ifi.Index)
	}
	gr.setGroup(grp)
	var p unsafe.Pointer
	var l sysSockoptLen
	if freebsd32o64 {
		var d [sysSizeofGroupReq + 4]byte
		s := (*[sysSizeofGroupReq]byte)(unsafe.Pointer(&gr))
		copy(d[:4], s[:4])
		copy(d[8:], s[4:])
		p = unsafe.Pointer(&d[0])
		l = sysSizeofGroupReq + 4
	} else {
		p = unsafe.Pointer(&gr)
		l = sysSizeofGroupReq
	}
	return os.NewSyscallError("setsockopt", setsockopt(fd, iana.ProtocolIP, name, p, l))
}

func setsockoptGroupSourceReq(fd, name int, ifi *net.Interface, grp, src net.IP) error {
	var gsr sysGroupSourceReq
	if ifi != nil {
		gsr.Interface = uint32(ifi.Index)
	}
	gsr.setSourceGroup(grp, src)
	var p unsafe.Pointer
	var l sysSockoptLen
	if freebsd32o64 {
		var d [sysSizeofGroupSourceReq + 4]byte
		s := (*[sysSizeofGroupSourceReq]byte)(unsafe.Pointer(&gsr))
		copy(d[:4], s[:4])
		copy(d[8:], s[4:])
		p = unsafe.Pointer(&d[0])
		l = sysSizeofGroupSourceReq + 4
	} else {
		p = unsafe.Pointer(&gsr)
		l = sysSizeofGroupSourceReq
	}
	return os.NewSyscallError("setsockopt", setsockopt(fd, iana.ProtocolIP, name, p, l))
}
