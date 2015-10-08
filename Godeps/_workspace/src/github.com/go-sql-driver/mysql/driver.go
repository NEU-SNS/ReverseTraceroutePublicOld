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
// Copyright 2012 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// The driver should be used via the database/sql package:
//
//  import "database/sql"
//  import _ "github.com/go-sql-driver/mysql"
//
//  db, err := sql.Open("mysql", "user:password@/dbname")
//
// See https://github.com/go-sql-driver/mysql#usage for details
package mysql

import (
	"database/sql"
	"database/sql/driver"
	"net"
)

// This struct is exported to make the driver directly accessible.
// In general the driver is used via the database/sql package.
type MySQLDriver struct{}

// DialFunc is a function which can be used to establish the network connection.
// Custom dial functions must be registered with RegisterDial
type DialFunc func(addr string) (net.Conn, error)

var dials map[string]DialFunc

// RegisterDial registers a custom dial function. It can then be used by the
// network address mynet(addr), where mynet is the registered new network.
// addr is passed as a parameter to the dial function.
func RegisterDial(net string, dial DialFunc) {
	if dials == nil {
		dials = make(map[string]DialFunc)
	}
	dials[net] = dial
}

// Open new Connection.
// See https://github.com/go-sql-driver/mysql#dsn-data-source-name for how
// the DSN string is formated
func (d MySQLDriver) Open(dsn string) (driver.Conn, error) {
	var err error

	// New mysqlConn
	mc := &mysqlConn{
		maxPacketAllowed: maxPacketSize,
		maxWriteSize:     maxPacketSize - 1,
	}
	mc.cfg, err = parseDSN(dsn)
	if err != nil {
		return nil, err
	}

	// Connect to Server
	if dial, ok := dials[mc.cfg.net]; ok {
		mc.netConn, err = dial(mc.cfg.addr)
	} else {
		nd := net.Dialer{Timeout: mc.cfg.timeout}
		mc.netConn, err = nd.Dial(mc.cfg.net, mc.cfg.addr)
	}
	if err != nil {
		return nil, err
	}

	// Enable TCP Keepalives on TCP connections
	if tc, ok := mc.netConn.(*net.TCPConn); ok {
		if err := tc.SetKeepAlive(true); err != nil {
			// Don't send COM_QUIT before handshake.
			mc.netConn.Close()
			mc.netConn = nil
			return nil, err
		}
	}

	mc.buf = newBuffer(mc.netConn)

	// Reading Handshake Initialization Packet
	cipher, err := mc.readInitPacket()
	if err != nil {
		mc.Close()
		return nil, err
	}

	// Send Client Authentication Packet
	if err = mc.writeAuthPacket(cipher); err != nil {
		mc.Close()
		return nil, err
	}

	// Read Result Packet
	err = mc.readResultOK()
	if err != nil {
		// Retry with old authentication method, if allowed
		if mc.cfg != nil && mc.cfg.allowOldPasswords && err == ErrOldPassword {
			if err = mc.writeOldAuthPacket(cipher); err != nil {
				mc.Close()
				return nil, err
			}
			if err = mc.readResultOK(); err != nil {
				mc.Close()
				return nil, err
			}
		} else if mc.cfg != nil && mc.cfg.allowCleartextPasswords && err == ErrCleartextPassword {
			if err = mc.writeClearAuthPacket(); err != nil {
				mc.Close()
				return nil, err
			}
			if err = mc.readResultOK(); err != nil {
				mc.Close()
				return nil, err
			}
		} else {
			mc.Close()
			return nil, err
		}

	}

	// Get max allowed packet size
	maxap, err := mc.getSystemVar("max_allowed_packet")
	if err != nil {
		mc.Close()
		return nil, err
	}
	mc.maxPacketAllowed = stringToInt(maxap) - 1
	if mc.maxPacketAllowed < maxPacketSize {
		mc.maxWriteSize = mc.maxPacketAllowed
	}

	// Handle DSN Params
	err = mc.handleParams()
	if err != nil {
		mc.Close()
		return nil, err
	}

	return mc, nil
}

func init() {
	sql.Register("mysql", &MySQLDriver{})
}
