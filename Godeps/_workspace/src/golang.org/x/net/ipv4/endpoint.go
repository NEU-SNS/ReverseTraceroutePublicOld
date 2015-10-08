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
	"syscall"
	"time"
)

// A Conn represents a network endpoint that uses the IPv4 transport.
// It is used to control basic IP-level socket options such as TOS and
// TTL.
type Conn struct {
	genericOpt
}

type genericOpt struct {
	net.Conn
}

func (c *genericOpt) ok() bool { return c != nil && c.Conn != nil }

// NewConn returns a new Conn.
func NewConn(c net.Conn) *Conn {
	return &Conn{
		genericOpt: genericOpt{Conn: c},
	}
}

// A PacketConn represents a packet network endpoint that uses the
// IPv4 transport.  It is used to control several IP-level socket
// options including multicasting.  It also provides datagram based
// network I/O methods specific to the IPv4 and higher layer protocols
// such as UDP.
type PacketConn struct {
	genericOpt
	dgramOpt
	payloadHandler
}

type dgramOpt struct {
	net.PacketConn
}

func (c *dgramOpt) ok() bool { return c != nil && c.PacketConn != nil }

// SetControlMessage sets the per packet IP-level socket options.
func (c *PacketConn) SetControlMessage(cf ControlFlags, on bool) error {
	if !c.payloadHandler.ok() {
		return syscall.EINVAL
	}
	fd, err := c.payloadHandler.sysfd()
	if err != nil {
		return err
	}
	return setControlMessage(fd, &c.payloadHandler.rawOpt, cf, on)
}

// SetDeadline sets the read and write deadlines associated with the
// endpoint.
func (c *PacketConn) SetDeadline(t time.Time) error {
	if !c.payloadHandler.ok() {
		return syscall.EINVAL
	}
	return c.payloadHandler.PacketConn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline associated with the
// endpoint.
func (c *PacketConn) SetReadDeadline(t time.Time) error {
	if !c.payloadHandler.ok() {
		return syscall.EINVAL
	}
	return c.payloadHandler.PacketConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline associated with the
// endpoint.
func (c *PacketConn) SetWriteDeadline(t time.Time) error {
	if !c.payloadHandler.ok() {
		return syscall.EINVAL
	}
	return c.payloadHandler.PacketConn.SetWriteDeadline(t)
}

// Close closes the endpoint.
func (c *PacketConn) Close() error {
	if !c.payloadHandler.ok() {
		return syscall.EINVAL
	}
	return c.payloadHandler.PacketConn.Close()
}

// NewPacketConn returns a new PacketConn using c as its underlying
// transport.
func NewPacketConn(c net.PacketConn) *PacketConn {
	p := &PacketConn{
		genericOpt:     genericOpt{Conn: c.(net.Conn)},
		dgramOpt:       dgramOpt{PacketConn: c},
		payloadHandler: payloadHandler{PacketConn: c},
	}
	if _, ok := c.(*net.IPConn); ok && sockOpts[ssoStripHeader].name > 0 {
		if fd, err := p.payloadHandler.sysfd(); err == nil {
			setInt(fd, &sockOpts[ssoStripHeader], boolint(true))
		}
	}
	return p
}

// A RawConn represents a packet network endpoint that uses the IPv4
// transport.  It is used to control several IP-level socket options
// including IPv4 header manipulation.  It also provides datagram
// based network I/O methods specific to the IPv4 and higher layer
// protocols that handle IPv4 datagram directly such as OSPF, GRE.
type RawConn struct {
	genericOpt
	dgramOpt
	packetHandler
}

// SetControlMessage sets the per packet IP-level socket options.
func (c *RawConn) SetControlMessage(cf ControlFlags, on bool) error {
	if !c.packetHandler.ok() {
		return syscall.EINVAL
	}
	fd, err := c.packetHandler.sysfd()
	if err != nil {
		return err
	}
	return setControlMessage(fd, &c.packetHandler.rawOpt, cf, on)
}

// SetDeadline sets the read and write deadlines associated with the
// endpoint.
func (c *RawConn) SetDeadline(t time.Time) error {
	if !c.packetHandler.ok() {
		return syscall.EINVAL
	}
	return c.packetHandler.c.SetDeadline(t)
}

// SetReadDeadline sets the read deadline associated with the
// endpoint.
func (c *RawConn) SetReadDeadline(t time.Time) error {
	if !c.packetHandler.ok() {
		return syscall.EINVAL
	}
	return c.packetHandler.c.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline associated with the
// endpoint.
func (c *RawConn) SetWriteDeadline(t time.Time) error {
	if !c.packetHandler.ok() {
		return syscall.EINVAL
	}
	return c.packetHandler.c.SetWriteDeadline(t)
}

// Close closes the endpoint.
func (c *RawConn) Close() error {
	if !c.packetHandler.ok() {
		return syscall.EINVAL
	}
	return c.packetHandler.c.Close()
}

// NewRawConn returns a new RawConn using c as its underlying
// transport.
func NewRawConn(c net.PacketConn) (*RawConn, error) {
	r := &RawConn{
		genericOpt:    genericOpt{Conn: c.(net.Conn)},
		dgramOpt:      dgramOpt{PacketConn: c},
		packetHandler: packetHandler{c: c.(*net.IPConn)},
	}
	fd, err := r.packetHandler.sysfd()
	if err != nil {
		return nil, err
	}
	if err := setInt(fd, &sockOpts[ssoHeaderPrepend], boolint(true)); err != nil {
		return nil, err
	}
	return r, nil
}
