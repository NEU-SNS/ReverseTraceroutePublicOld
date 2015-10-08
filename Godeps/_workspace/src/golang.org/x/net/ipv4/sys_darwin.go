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

package ipv4

import (
	"net"
	"syscall"
	"unsafe"
)

type sysSockoptLen int32

var (
	ctlOpts = [ctlMax]ctlOpt{
		ctlTTL:       {sysIP_RECVTTL, 1, marshalTTL, parseTTL},
		ctlDst:       {sysIP_RECVDSTADDR, net.IPv4len, marshalDst, parseDst},
		ctlInterface: {sysIP_RECVIF, syscall.SizeofSockaddrDatalink, marshalInterface, parseInterface},
	}

	sockOpts = [ssoMax]sockOpt{
		ssoTOS:                {sysIP_TOS, ssoTypeInt},
		ssoTTL:                {sysIP_TTL, ssoTypeInt},
		ssoMulticastTTL:       {sysIP_MULTICAST_TTL, ssoTypeByte},
		ssoMulticastInterface: {sysIP_MULTICAST_IF, ssoTypeInterface},
		ssoMulticastLoopback:  {sysIP_MULTICAST_LOOP, ssoTypeInt},
		ssoReceiveTTL:         {sysIP_RECVTTL, ssoTypeInt},
		ssoReceiveDst:         {sysIP_RECVDSTADDR, ssoTypeInt},
		ssoReceiveInterface:   {sysIP_RECVIF, ssoTypeInt},
		ssoHeaderPrepend:      {sysIP_HDRINCL, ssoTypeInt},
		ssoStripHeader:        {sysIP_STRIPHDR, ssoTypeInt},
		ssoJoinGroup:          {sysIP_ADD_MEMBERSHIP, ssoTypeIPMreq},
		ssoLeaveGroup:         {sysIP_DROP_MEMBERSHIP, ssoTypeIPMreq},
	}
)

func init() {
	// Seems like kern.osreldate is veiled on latest OS X. We use
	// kern.osrelease instead.
	osver, err := syscall.Sysctl("kern.osrelease")
	if err != nil {
		return
	}
	var i int
	for i = range osver {
		if osver[i] == '.' {
			break
		}
	}
	// The IP_PKTINFO and protocol-independent multicast API were
	// introduced in OS X 10.7 (Darwin 11.0.0). But it looks like
	// those features require OS X 10.8 (Darwin 12.0.0) and above.
	// See http://support.apple.com/kb/HT1633.
	if i > 2 || i == 2 && osver[0] >= '1' && osver[1] >= '2' {
		ctlOpts[ctlPacketInfo].name = sysIP_PKTINFO
		ctlOpts[ctlPacketInfo].length = sysSizeofInetPktinfo
		ctlOpts[ctlPacketInfo].marshal = marshalPacketInfo
		ctlOpts[ctlPacketInfo].parse = parsePacketInfo
		sockOpts[ssoPacketInfo].name = sysIP_RECVPKTINFO
		sockOpts[ssoPacketInfo].typ = ssoTypeInt
		sockOpts[ssoMulticastInterface].typ = ssoTypeIPMreqn
		sockOpts[ssoJoinGroup].name = sysMCAST_JOIN_GROUP
		sockOpts[ssoJoinGroup].typ = ssoTypeGroupReq
		sockOpts[ssoLeaveGroup].name = sysMCAST_LEAVE_GROUP
		sockOpts[ssoLeaveGroup].typ = ssoTypeGroupReq
		sockOpts[ssoJoinSourceGroup].name = sysMCAST_JOIN_SOURCE_GROUP
		sockOpts[ssoJoinSourceGroup].typ = ssoTypeGroupSourceReq
		sockOpts[ssoLeaveSourceGroup].name = sysMCAST_LEAVE_SOURCE_GROUP
		sockOpts[ssoLeaveSourceGroup].typ = ssoTypeGroupSourceReq
		sockOpts[ssoBlockSourceGroup].name = sysMCAST_BLOCK_SOURCE
		sockOpts[ssoBlockSourceGroup].typ = ssoTypeGroupSourceReq
		sockOpts[ssoUnblockSourceGroup].name = sysMCAST_UNBLOCK_SOURCE
		sockOpts[ssoUnblockSourceGroup].typ = ssoTypeGroupSourceReq
	}
}

func (pi *sysInetPktinfo) setIfindex(i int) {
	pi.Ifindex = uint32(i)
}

func (gr *sysGroupReq) setGroup(grp net.IP) {
	sa := (*sysSockaddrInet)(unsafe.Pointer(&gr.Pad_cgo_0[0]))
	sa.Len = sysSizeofSockaddrInet
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], grp)
}

func (gsr *sysGroupSourceReq) setSourceGroup(grp, src net.IP) {
	sa := (*sysSockaddrInet)(unsafe.Pointer(&gsr.Pad_cgo_0[0]))
	sa.Len = sysSizeofSockaddrInet
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], grp)
	sa = (*sysSockaddrInet)(unsafe.Pointer(&gsr.Pad_cgo_1[0]))
	sa.Len = sysSizeofSockaddrInet
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], src)
}
