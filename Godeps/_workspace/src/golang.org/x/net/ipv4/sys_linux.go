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
		ctlTTL:        {sysIP_TTL, 1, marshalTTL, parseTTL},
		ctlPacketInfo: {sysIP_PKTINFO, sysSizeofInetPktinfo, marshalPacketInfo, parsePacketInfo},
	}

	sockOpts = [ssoMax]sockOpt{
		ssoTOS:                {sysIP_TOS, ssoTypeInt},
		ssoTTL:                {sysIP_TTL, ssoTypeInt},
		ssoMulticastTTL:       {sysIP_MULTICAST_TTL, ssoTypeInt},
		ssoMulticastInterface: {sysIP_MULTICAST_IF, ssoTypeIPMreqn},
		ssoMulticastLoopback:  {sysIP_MULTICAST_LOOP, ssoTypeInt},
		ssoReceiveTTL:         {sysIP_RECVTTL, ssoTypeInt},
		ssoPacketInfo:         {sysIP_PKTINFO, ssoTypeInt},
		ssoHeaderPrepend:      {sysIP_HDRINCL, ssoTypeInt},
		ssoICMPFilter:         {sysICMP_FILTER, ssoTypeICMPFilter},
		ssoJoinGroup:          {sysMCAST_JOIN_GROUP, ssoTypeGroupReq},
		ssoLeaveGroup:         {sysMCAST_LEAVE_GROUP, ssoTypeGroupReq},
		ssoJoinSourceGroup:    {sysMCAST_JOIN_SOURCE_GROUP, ssoTypeGroupSourceReq},
		ssoLeaveSourceGroup:   {sysMCAST_LEAVE_SOURCE_GROUP, ssoTypeGroupSourceReq},
		ssoBlockSourceGroup:   {sysMCAST_BLOCK_SOURCE, ssoTypeGroupSourceReq},
		ssoUnblockSourceGroup: {sysMCAST_UNBLOCK_SOURCE, ssoTypeGroupSourceReq},
	}
)

func (pi *sysInetPktinfo) setIfindex(i int) {
	pi.Ifindex = int32(i)
}

func (gr *sysGroupReq) setGroup(grp net.IP) {
	sa := (*sysSockaddrInet)(unsafe.Pointer(&gr.Group))
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], grp)
}

func (gsr *sysGroupSourceReq) setSourceGroup(grp, src net.IP) {
	sa := (*sysSockaddrInet)(unsafe.Pointer(&gsr.Group))
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], grp)
	sa = (*sysSockaddrInet)(unsafe.Pointer(&gsr.Source))
	sa.Family = syscall.AF_INET
	copy(sa.Addr[:], src)
}
