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

// +build darwin dragonfly freebsd linux netbsd openbsd

package ipv4

import (
	"net"
	"os"
	"unsafe"

	"golang.org/x/net/internal/iana"
)

func getInt(fd int, opt *sockOpt) (int, error) {
	if opt.name < 1 || (opt.typ != ssoTypeByte && opt.typ != ssoTypeInt) {
		return 0, errOpNoSupport
	}
	var i int32
	var b byte
	p := unsafe.Pointer(&i)
	l := sysSockoptLen(4)
	if opt.typ == ssoTypeByte {
		p = unsafe.Pointer(&b)
		l = sysSockoptLen(1)
	}
	if err := getsockopt(fd, iana.ProtocolIP, opt.name, p, &l); err != nil {
		return 0, os.NewSyscallError("getsockopt", err)
	}
	if opt.typ == ssoTypeByte {
		return int(b), nil
	}
	return int(i), nil
}

func setInt(fd int, opt *sockOpt, v int) error {
	if opt.name < 1 || (opt.typ != ssoTypeByte && opt.typ != ssoTypeInt) {
		return errOpNoSupport
	}
	i := int32(v)
	var b byte
	p := unsafe.Pointer(&i)
	l := sysSockoptLen(4)
	if opt.typ == ssoTypeByte {
		b = byte(v)
		p = unsafe.Pointer(&b)
		l = sysSockoptLen(1)
	}
	return os.NewSyscallError("setsockopt", setsockopt(fd, iana.ProtocolIP, opt.name, p, l))
}

func getInterface(fd int, opt *sockOpt) (*net.Interface, error) {
	if opt.name < 1 {
		return nil, errOpNoSupport
	}
	switch opt.typ {
	case ssoTypeInterface:
		return getsockoptInterface(fd, opt.name)
	case ssoTypeIPMreqn:
		return getsockoptIPMreqn(fd, opt.name)
	default:
		return nil, errOpNoSupport
	}
}

func setInterface(fd int, opt *sockOpt, ifi *net.Interface) error {
	if opt.name < 1 {
		return errOpNoSupport
	}
	switch opt.typ {
	case ssoTypeInterface:
		return setsockoptInterface(fd, opt.name, ifi)
	case ssoTypeIPMreqn:
		return setsockoptIPMreqn(fd, opt.name, ifi, nil)
	default:
		return errOpNoSupport
	}
}

func getICMPFilter(fd int, opt *sockOpt) (*ICMPFilter, error) {
	if opt.name < 1 || opt.typ != ssoTypeICMPFilter {
		return nil, errOpNoSupport
	}
	var f ICMPFilter
	l := sysSockoptLen(sysSizeofICMPFilter)
	if err := getsockopt(fd, iana.ProtocolReserved, opt.name, unsafe.Pointer(&f.sysICMPFilter), &l); err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	return &f, nil
}

func setICMPFilter(fd int, opt *sockOpt, f *ICMPFilter) error {
	if opt.name < 1 || opt.typ != ssoTypeICMPFilter {
		return errOpNoSupport
	}
	return os.NewSyscallError("setsockopt", setsockopt(fd, iana.ProtocolReserved, opt.name, unsafe.Pointer(&f.sysICMPFilter), sysSizeofICMPFilter))
}

func setGroup(fd int, opt *sockOpt, ifi *net.Interface, grp net.IP) error {
	if opt.name < 1 {
		return errOpNoSupport
	}
	switch opt.typ {
	case ssoTypeIPMreq:
		return setsockoptIPMreq(fd, opt.name, ifi, grp)
	case ssoTypeIPMreqn:
		return setsockoptIPMreqn(fd, opt.name, ifi, grp)
	case ssoTypeGroupReq:
		return setsockoptGroupReq(fd, opt.name, ifi, grp)
	default:
		return errOpNoSupport
	}
}

func setSourceGroup(fd int, opt *sockOpt, ifi *net.Interface, grp, src net.IP) error {
	if opt.name < 1 || opt.typ != ssoTypeGroupSourceReq {
		return errOpNoSupport
	}
	return setsockoptGroupSourceReq(fd, opt.name, ifi, grp, src)
}
