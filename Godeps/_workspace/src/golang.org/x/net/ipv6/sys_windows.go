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
// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ipv6

import (
	"net"
	"syscall"

	"golang.org/x/net/internal/iana"
)

const (
	// See ws2tcpip.h.
	sysIPV6_UNICAST_HOPS   = 0x4
	sysIPV6_MULTICAST_IF   = 0x9
	sysIPV6_MULTICAST_HOPS = 0xa
	sysIPV6_MULTICAST_LOOP = 0xb
	sysIPV6_JOIN_GROUP     = 0xc
	sysIPV6_LEAVE_GROUP    = 0xd
	sysIPV6_PKTINFO        = 0x13

	sysSizeofSockaddrInet6 = 0x1c

	sysSizeofIPv6Mreq = 0x14
)

type sysSockaddrInet6 struct {
	Family   uint16
	Port     uint16
	Flowinfo uint32
	Addr     [16]byte /* in6_addr */
	Scope_id uint32
}

type sysIPv6Mreq struct {
	Multiaddr [16]byte /* in6_addr */
	Interface uint32
}

var (
	ctlOpts = [ctlMax]ctlOpt{}

	sockOpts = [ssoMax]sockOpt{
		ssoHopLimit:           {iana.ProtocolIPv6, sysIPV6_UNICAST_HOPS, ssoTypeInt},
		ssoMulticastInterface: {iana.ProtocolIPv6, sysIPV6_MULTICAST_IF, ssoTypeInterface},
		ssoMulticastHopLimit:  {iana.ProtocolIPv6, sysIPV6_MULTICAST_HOPS, ssoTypeInt},
		ssoMulticastLoopback:  {iana.ProtocolIPv6, sysIPV6_MULTICAST_LOOP, ssoTypeInt},
		ssoJoinGroup:          {iana.ProtocolIPv6, sysIPV6_JOIN_GROUP, ssoTypeIPMreq},
		ssoLeaveGroup:         {iana.ProtocolIPv6, sysIPV6_LEAVE_GROUP, ssoTypeIPMreq},
	}
)

func (sa *sysSockaddrInet6) setSockaddr(ip net.IP, i int) {
	sa.Family = syscall.AF_INET6
	copy(sa.Addr[:], ip)
	sa.Scope_id = uint32(i)
}

func (mreq *sysIPv6Mreq) setIfindex(i int) {
	mreq.Interface = uint32(i)
}
