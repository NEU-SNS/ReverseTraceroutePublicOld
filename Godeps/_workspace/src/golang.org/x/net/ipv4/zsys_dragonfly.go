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
// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs defs_dragonfly.go

// +build dragonfly

package ipv4

const (
	sysIP_OPTIONS     = 0x1
	sysIP_HDRINCL     = 0x2
	sysIP_TOS         = 0x3
	sysIP_TTL         = 0x4
	sysIP_RECVOPTS    = 0x5
	sysIP_RECVRETOPTS = 0x6
	sysIP_RECVDSTADDR = 0x7
	sysIP_RETOPTS     = 0x8
	sysIP_RECVIF      = 0x14
	sysIP_RECVTTL     = 0x41

	sysIP_MULTICAST_IF    = 0x9
	sysIP_MULTICAST_TTL   = 0xa
	sysIP_MULTICAST_LOOP  = 0xb
	sysIP_MULTICAST_VIF   = 0xe
	sysIP_ADD_MEMBERSHIP  = 0xc
	sysIP_DROP_MEMBERSHIP = 0xd

	sysSizeofIPMreq = 0x8
)

type sysIPMreq struct {
	Multiaddr [4]byte /* in_addr */
	Interface [4]byte /* in_addr */
}
