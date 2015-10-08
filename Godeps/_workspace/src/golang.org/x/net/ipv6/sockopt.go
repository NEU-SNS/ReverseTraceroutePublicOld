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

package ipv6

// Sticky socket options
const (
	ssoTrafficClass        = iota // header field for unicast packet, RFC 3542
	ssoHopLimit                   // header field for unicast packet, RFC 3493
	ssoMulticastInterface         // outbound interface for multicast packet, RFC 3493
	ssoMulticastHopLimit          // header field for multicast packet, RFC 3493
	ssoMulticastLoopback          // loopback for multicast packet, RFC 3493
	ssoReceiveTrafficClass        // header field on received packet, RFC 3542
	ssoReceiveHopLimit            // header field on received packet, RFC 2292 or 3542
	ssoReceivePacketInfo          // incbound or outbound packet path, RFC 2292 or 3542
	ssoReceivePathMTU             // path mtu, RFC 3542
	ssoPathMTU                    // path mtu, RFC 3542
	ssoChecksum                   // packet checksum, RFC 2292 or 3542
	ssoICMPFilter                 // icmp filter, RFC 2292 or 3542
	ssoJoinGroup                  // any-source multicast, RFC 3493
	ssoLeaveGroup                 // any-source multicast, RFC 3493
	ssoJoinSourceGroup            // source-specific multicast
	ssoLeaveSourceGroup           // source-specific multicast
	ssoBlockSourceGroup           // any-source or source-specific multicast
	ssoUnblockSourceGroup         // any-source or source-specific multicast
	ssoMax
)

// Sticky socket option value types
const (
	ssoTypeInt = iota + 1
	ssoTypeInterface
	ssoTypeICMPFilter
	ssoTypeMTUInfo
	ssoTypeIPMreq
	ssoTypeGroupReq
	ssoTypeGroupSourceReq
)

// A sockOpt represents a binding for sticky socket option.
type sockOpt struct {
	level int // option level
	name  int // option name, must be equal or greater than 1
	typ   int // option value type, must be equal or greater than 1
}
