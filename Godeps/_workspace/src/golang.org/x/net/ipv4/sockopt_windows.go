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

package ipv4

import (
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/net/internal/iana"
)

func getInt(fd syscall.Handle, opt *sockOpt) (int, error) {
	if opt.name < 1 || opt.typ != ssoTypeInt {
		return 0, errOpNoSupport
	}
	var i int32
	l := int32(4)
	if err := syscall.Getsockopt(fd, iana.ProtocolIP, int32(opt.name), (*byte)(unsafe.Pointer(&i)), &l); err != nil {
		return 0, os.NewSyscallError("getsockopt", err)
	}
	return int(i), nil
}

func setInt(fd syscall.Handle, opt *sockOpt, v int) error {
	if opt.name < 1 || opt.typ != ssoTypeInt {
		return errOpNoSupport
	}
	i := int32(v)
	return os.NewSyscallError("setsockopt", syscall.Setsockopt(fd, iana.ProtocolIP, int32(opt.name), (*byte)(unsafe.Pointer(&i)), 4))
}

func getInterface(fd syscall.Handle, opt *sockOpt) (*net.Interface, error) {
	if opt.name < 1 || opt.typ != ssoTypeInterface {
		return nil, errOpNoSupport
	}
	return getsockoptInterface(fd, opt.name)
}

func setInterface(fd syscall.Handle, opt *sockOpt, ifi *net.Interface) error {
	if opt.name < 1 || opt.typ != ssoTypeInterface {
		return errOpNoSupport
	}
	return setsockoptInterface(fd, opt.name, ifi)
}

func getICMPFilter(fd syscall.Handle, opt *sockOpt) (*ICMPFilter, error) {
	return nil, errOpNoSupport
}

func setICMPFilter(fd syscall.Handle, opt *sockOpt, f *ICMPFilter) error {
	return errOpNoSupport
}

func setGroup(fd syscall.Handle, opt *sockOpt, ifi *net.Interface, grp net.IP) error {
	if opt.name < 1 || opt.typ != ssoTypeIPMreq {
		return errOpNoSupport
	}
	return setsockoptIPMreq(fd, opt.name, ifi, grp)
}

func setSourceGroup(fd syscall.Handle, opt *sockOpt, ifi *net.Interface, grp, src net.IP) error {
	// TODO(mikio): implement this
	return errOpNoSupport
}
