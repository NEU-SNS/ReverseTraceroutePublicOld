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

// +build ignore

// +godefs map struct_in_addr [4]byte /* in_addr */

package ipv4

/*
#include <netinet/in.h>
*/
import "C"

const (
	sysIP_OPTIONS     = C.IP_OPTIONS
	sysIP_HDRINCL     = C.IP_HDRINCL
	sysIP_TOS         = C.IP_TOS
	sysIP_TTL         = C.IP_TTL
	sysIP_RECVOPTS    = C.IP_RECVOPTS
	sysIP_RECVRETOPTS = C.IP_RECVRETOPTS
	sysIP_RECVDSTADDR = C.IP_RECVDSTADDR
	sysIP_RETOPTS     = C.IP_RETOPTS
	sysIP_RECVIF      = C.IP_RECVIF
	sysIP_RECVTTL     = C.IP_RECVTTL

	sysIP_MULTICAST_IF    = C.IP_MULTICAST_IF
	sysIP_MULTICAST_TTL   = C.IP_MULTICAST_TTL
	sysIP_MULTICAST_LOOP  = C.IP_MULTICAST_LOOP
	sysIP_MULTICAST_VIF   = C.IP_MULTICAST_VIF
	sysIP_ADD_MEMBERSHIP  = C.IP_ADD_MEMBERSHIP
	sysIP_DROP_MEMBERSHIP = C.IP_DROP_MEMBERSHIP

	sysSizeofIPMreq = C.sizeof_struct_ip_mreq
)

type sysIPMreq C.struct_ip_mreq
