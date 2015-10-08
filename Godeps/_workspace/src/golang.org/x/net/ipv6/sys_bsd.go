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

// +build dragonfly netbsd openbsd

package ipv6

import (
	"net"
	"syscall"

	"golang.org/x/net/internal/iana"
)

type sysSockoptLen int32

var (
	ctlOpts = [ctlMax]ctlOpt{
		ctlTrafficClass: {sysIPV6_TCLASS, 4, marshalTrafficClass, parseTrafficClass},
		ctlHopLimit:     {sysIPV6_HOPLIMIT, 4, marshalHopLimit, parseHopLimit},
		ctlPacketInfo:   {sysIPV6_PKTINFO, sysSizeofInet6Pktinfo, marshalPacketInfo, parsePacketInfo},
		ctlNextHop:      {sysIPV6_NEXTHOP, sysSizeofSockaddrInet6, marshalNextHop, parseNextHop},
		ctlPathMTU:      {sysIPV6_PATHMTU, sysSizeofIPv6Mtuinfo, marshalPathMTU, parsePathMTU},
	}

	sockOpts = [ssoMax]sockOpt{
		ssoTrafficClass:        {iana.ProtocolIPv6, sysIPV6_TCLASS, ssoTypeInt},
		ssoHopLimit:            {iana.ProtocolIPv6, sysIPV6_UNICAST_HOPS, ssoTypeInt},
		ssoMulticastInterface:  {iana.ProtocolIPv6, sysIPV6_MULTICAST_IF, ssoTypeInterface},
		ssoMulticastHopLimit:   {iana.ProtocolIPv6, sysIPV6_MULTICAST_HOPS, ssoTypeInt},
		ssoMulticastLoopback:   {iana.ProtocolIPv6, sysIPV6_MULTICAST_LOOP, ssoTypeInt},
		ssoReceiveTrafficClass: {iana.ProtocolIPv6, sysIPV6_RECVTCLASS, ssoTypeInt},
		ssoReceiveHopLimit:     {iana.ProtocolIPv6, sysIPV6_RECVHOPLIMIT, ssoTypeInt},
		ssoReceivePacketInfo:   {iana.ProtocolIPv6, sysIPV6_RECVPKTINFO, ssoTypeInt},
		ssoReceivePathMTU:      {iana.ProtocolIPv6, sysIPV6_RECVPATHMTU, ssoTypeInt},
		ssoPathMTU:             {iana.ProtocolIPv6, sysIPV6_PATHMTU, ssoTypeMTUInfo},
		ssoChecksum:            {iana.ProtocolIPv6, sysIPV6_CHECKSUM, ssoTypeInt},
		ssoICMPFilter:          {iana.ProtocolIPv6ICMP, sysICMP6_FILTER, ssoTypeICMPFilter},
		ssoJoinGroup:           {iana.ProtocolIPv6, sysIPV6_JOIN_GROUP, ssoTypeIPMreq},
		ssoLeaveGroup:          {iana.ProtocolIPv6, sysIPV6_LEAVE_GROUP, ssoTypeIPMreq},
	}
)

func (sa *sysSockaddrInet6) setSockaddr(ip net.IP, i int) {
	sa.Len = sysSizeofSockaddrInet6
	sa.Family = syscall.AF_INET6
	copy(sa.Addr[:], ip)
	sa.Scope_id = uint32(i)
}

func (pi *sysInet6Pktinfo) setIfindex(i int) {
	pi.Ifindex = uint32(i)
}

func (mreq *sysIPv6Mreq) setIfindex(i int) {
	mreq.Interface = uint32(i)
}
