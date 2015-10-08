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

// Sticky socket options
const (
	ssoTOS                = iota // header field for unicast packet
	ssoTTL                       // header field for unicast packet
	ssoMulticastTTL              // header field for multicast packet
	ssoMulticastInterface        // outbound interface for multicast packet
	ssoMulticastLoopback         // loopback for multicast packet
	ssoReceiveTTL                // header field on received packet
	ssoReceiveDst                // header field on received packet
	ssoReceiveInterface          // inbound interface on received packet
	ssoPacketInfo                // incbound or outbound packet path
	ssoHeaderPrepend             // ipv4 header prepend
	ssoStripHeader               // strip ipv4 header
	ssoICMPFilter                // icmp filter
	ssoJoinGroup                 // any-source multicast
	ssoLeaveGroup                // any-source multicast
	ssoJoinSourceGroup           // source-specific multicast
	ssoLeaveSourceGroup          // source-specific multicast
	ssoBlockSourceGroup          // any-source or source-specific multicast
	ssoUnblockSourceGroup        // any-source or source-specific multicast
	ssoMax
)

// Sticky socket option value types
const (
	ssoTypeByte = iota + 1
	ssoTypeInt
	ssoTypeInterface
	ssoTypeICMPFilter
	ssoTypeIPMreq
	ssoTypeIPMreqn
	ssoTypeGroupReq
	ssoTypeGroupSourceReq
)

// A sockOpt represents a binding for sticky socket option.
type sockOpt struct {
	name int // option name, must be equal or greater than 1
	typ  int // option value type, must be equal or greater than 1
}
