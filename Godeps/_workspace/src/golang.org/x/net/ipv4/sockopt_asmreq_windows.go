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

func setsockoptIPMreq(fd syscall.Handle, name int, ifi *net.Interface, grp net.IP) error {
	mreq := sysIPMreq{Multiaddr: [4]byte{grp[0], grp[1], grp[2], grp[3]}}
	if err := setIPMreqInterface(&mreq, ifi); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", syscall.Setsockopt(fd, iana.ProtocolIP, int32(name), (*byte)(unsafe.Pointer(&mreq)), int32(sysSizeofIPMreq)))
}

func getsockoptInterface(fd syscall.Handle, name int) (*net.Interface, error) {
	var b [4]byte
	l := int32(4)
	if err := syscall.Getsockopt(fd, iana.ProtocolIP, int32(name), (*byte)(unsafe.Pointer(&b[0])), &l); err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	ifi, err := netIP4ToInterface(net.IPv4(b[0], b[1], b[2], b[3]))
	if err != nil {
		return nil, err
	}
	return ifi, nil
}

func setsockoptInterface(fd syscall.Handle, name int, ifi *net.Interface) error {
	ip, err := netInterfaceToIP4(ifi)
	if err != nil {
		return err
	}
	var b [4]byte
	copy(b[:], ip)
	return os.NewSyscallError("setsockopt", syscall.Setsockopt(fd, iana.ProtocolIP, int32(name), (*byte)(unsafe.Pointer(&b[0])), 4))
}
