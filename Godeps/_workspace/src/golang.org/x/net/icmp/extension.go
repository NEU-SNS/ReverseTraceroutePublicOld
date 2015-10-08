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
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icmp

// An Extension represents an ICMP extension.
type Extension interface {
	// Len returns the length of ICMP extension.
	// Proto must be either the ICMPv4 or ICMPv6 protocol number.
	Len(proto int) int

	// Marshal returns the binary enconding of ICMP extension.
	// Proto must be either the ICMPv4 or ICMPv6 protocol number.
	Marshal(proto int) ([]byte, error)
}

const extensionVersion = 2

func validExtensionHeader(b []byte) bool {
	v := int(b[0]&0xf0) >> 4
	s := uint16(b[2])<<8 | uint16(b[3])
	if s != 0 {
		s = checksum(b)
	}
	if v != extensionVersion || s != 0 {
		return false
	}
	return true
}

// parseExtensions parses b as a list of ICMP extensions.
// The length attribute l must be the length attribute field in
// received icmp messages.
//
// It will return a list of ICMP extensions and an adjusted length
// attribute that represents the length of the padded original
// datagram field. Otherwise, it returns an error.
func parseExtensions(b []byte, l int) ([]Extension, int, error) {
	// Still a lot of non-RFC 4884 compliant implementations are
	// out there. Set the length attribute l to 128 when it looks
	// inappropriate for backwards compatibility.
	//
	// A minimal extension at least requires 8 octets; 4 octets
	// for an extension header, and 4 octets for a single object
	// header.
	//
	// See RFC 4884 for further information.
	if 128 > l || l+8 > len(b) {
		l = 128
	}
	if l+8 > len(b) {
		return nil, -1, errNoExtension
	}
	if !validExtensionHeader(b[l:]) {
		if l == 128 {
			return nil, -1, errNoExtension
		}
		l = 128
		if !validExtensionHeader(b[l:]) {
			return nil, -1, errNoExtension
		}
	}
	var exts []Extension
	for b = b[l+4:]; len(b) >= 4; {
		ol := int(b[0])<<8 | int(b[1])
		if 4 > ol || ol > len(b) {
			break
		}
		switch b[2] {
		case classMPLSLabelStack:
			ext, err := parseMPLSLabelStack(b[:ol])
			if err != nil {
				return nil, -1, err
			}
			exts = append(exts, ext)
		case classInterfaceInfo:
			ext, err := parseInterfaceInfo(b[:ol])
			if err != nil {
				return nil, -1, err
			}
			exts = append(exts, ext)
		}
		b = b[ol:]
	}
	return exts, l, nil
}
