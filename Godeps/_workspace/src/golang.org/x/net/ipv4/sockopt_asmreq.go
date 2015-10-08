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

// +build darwin dragonfly freebsd netbsd openbsd windows

package ipv4

import "net"

func setIPMreqInterface(mreq *sysIPMreq, ifi *net.Interface) error {
	if ifi == nil {
		return nil
	}
	ifat, err := ifi.Addrs()
	if err != nil {
		return err
	}
	for _, ifa := range ifat {
		switch ifa := ifa.(type) {
		case *net.IPAddr:
			if ip := ifa.IP.To4(); ip != nil {
				copy(mreq.Interface[:], ip)
				return nil
			}
		case *net.IPNet:
			if ip := ifa.IP.To4(); ip != nil {
				copy(mreq.Interface[:], ip)
				return nil
			}
		}
	}
	return errNoSuchInterface
}

func netIP4ToInterface(ip net.IP) (*net.Interface, error) {
	ift, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifi := range ift {
		ifat, err := ifi.Addrs()
		if err != nil {
			return nil, err
		}
		for _, ifa := range ifat {
			switch ifa := ifa.(type) {
			case *net.IPAddr:
				if ip.Equal(ifa.IP) {
					return &ifi, nil
				}
			case *net.IPNet:
				if ip.Equal(ifa.IP) {
					return &ifi, nil
				}
			}
		}
	}
	return nil, errNoSuchInterface
}

func netInterfaceToIP4(ifi *net.Interface) (net.IP, error) {
	if ifi == nil {
		return net.IPv4zero.To4(), nil
	}
	ifat, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, ifa := range ifat {
		switch ifa := ifa.(type) {
		case *net.IPAddr:
			if ip := ifa.IP.To4(); ip != nil {
				return ip, nil
			}
		case *net.IPNet:
			if ip := ifa.IP.To4(); ip != nil {
				return ip, nil
			}
		}
	}
	return nil, errNoSuchInterface
}
