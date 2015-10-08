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
// cgo -godefs defs_openbsd.go

package ipv6

const (
	sysIPV6_UNICAST_HOPS   = 0x4
	sysIPV6_MULTICAST_IF   = 0x9
	sysIPV6_MULTICAST_HOPS = 0xa
	sysIPV6_MULTICAST_LOOP = 0xb
	sysIPV6_JOIN_GROUP     = 0xc
	sysIPV6_LEAVE_GROUP    = 0xd
	sysIPV6_PORTRANGE      = 0xe
	sysICMP6_FILTER        = 0x12

	sysIPV6_CHECKSUM = 0x1a
	sysIPV6_V6ONLY   = 0x1b

	sysIPV6_RTHDRDSTOPTS = 0x23

	sysIPV6_RECVPKTINFO  = 0x24
	sysIPV6_RECVHOPLIMIT = 0x25
	sysIPV6_RECVRTHDR    = 0x26
	sysIPV6_RECVHOPOPTS  = 0x27
	sysIPV6_RECVDSTOPTS  = 0x28

	sysIPV6_USE_MIN_MTU = 0x2a
	sysIPV6_RECVPATHMTU = 0x2b

	sysIPV6_PATHMTU = 0x2c

	sysIPV6_PKTINFO  = 0x2e
	sysIPV6_HOPLIMIT = 0x2f
	sysIPV6_NEXTHOP  = 0x30
	sysIPV6_HOPOPTS  = 0x31
	sysIPV6_DSTOPTS  = 0x32
	sysIPV6_RTHDR    = 0x33

	sysIPV6_AUTH_LEVEL        = 0x35
	sysIPV6_ESP_TRANS_LEVEL   = 0x36
	sysIPV6_ESP_NETWORK_LEVEL = 0x37
	sysIPSEC6_OUTSA           = 0x38
	sysIPV6_RECVTCLASS        = 0x39

	sysIPV6_AUTOFLOWLABEL = 0x3b
	sysIPV6_IPCOMP_LEVEL  = 0x3c

	sysIPV6_TCLASS   = 0x3d
	sysIPV6_DONTFRAG = 0x3e
	sysIPV6_PIPEX    = 0x3f

	sysIPV6_RTABLE = 0x1021

	sysIPV6_PORTRANGE_DEFAULT = 0x0
	sysIPV6_PORTRANGE_HIGH    = 0x1
	sysIPV6_PORTRANGE_LOW     = 0x2

	sysSizeofSockaddrInet6 = 0x1c
	sysSizeofInet6Pktinfo  = 0x14
	sysSizeofIPv6Mtuinfo   = 0x20

	sysSizeofIPv6Mreq = 0x14

	sysSizeofICMPv6Filter = 0x20
)

type sysSockaddrInet6 struct {
	Len      uint8
	Family   uint8
	Port     uint16
	Flowinfo uint32
	Addr     [16]byte /* in6_addr */
	Scope_id uint32
}

type sysInet6Pktinfo struct {
	Addr    [16]byte /* in6_addr */
	Ifindex uint32
}

type sysIPv6Mtuinfo struct {
	Addr sysSockaddrInet6
	Mtu  uint32
}

type sysIPv6Mreq struct {
	Multiaddr [16]byte /* in6_addr */
	Interface uint32
}

type sysICMPv6Filter struct {
	Filt [8]uint32
}
