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
// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2013 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

import "io"

const defaultBufSize = 4096

// A buffer which is used for both reading and writing.
// This is possible since communication on each connection is synchronous.
// In other words, we can't write and read simultaneously on the same connection.
// The buffer is similar to bufio.Reader / Writer but zero-copy-ish
// Also highly optimized for this particular use case.
type buffer struct {
	buf    []byte
	rd     io.Reader
	idx    int
	length int
}

func newBuffer(rd io.Reader) buffer {
	var b [defaultBufSize]byte
	return buffer{
		buf: b[:],
		rd:  rd,
	}
}

// fill reads into the buffer until at least _need_ bytes are in it
func (b *buffer) fill(need int) error {
	n := b.length

	// move existing data to the beginning
	if n > 0 && b.idx > 0 {
		copy(b.buf[0:n], b.buf[b.idx:])
	}

	// grow buffer if necessary
	// TODO: let the buffer shrink again at some point
	//       Maybe keep the org buf slice and swap back?
	if need > len(b.buf) {
		// Round up to the next multiple of the default size
		newBuf := make([]byte, ((need/defaultBufSize)+1)*defaultBufSize)
		copy(newBuf, b.buf)
		b.buf = newBuf
	}

	b.idx = 0

	for {
		nn, err := b.rd.Read(b.buf[n:])
		n += nn

		switch err {
		case nil:
			if n < need {
				continue
			}
			b.length = n
			return nil

		case io.EOF:
			if n >= need {
				b.length = n
				return nil
			}
			return io.ErrUnexpectedEOF

		default:
			return err
		}
	}
}

// returns next N bytes from buffer.
// The returned slice is only guaranteed to be valid until the next read
func (b *buffer) readNext(need int) ([]byte, error) {
	if b.length < need {
		// refill
		if err := b.fill(need); err != nil {
			return nil, err
		}
	}

	offset := b.idx
	b.idx += need
	b.length -= need
	return b.buf[offset:b.idx], nil
}

// returns a buffer with the requested size.
// If possible, a slice from the existing buffer is returned.
// Otherwise a bigger buffer is made.
// Only one buffer (total) can be used at a time.
func (b *buffer) takeBuffer(length int) []byte {
	if b.length > 0 {
		return nil
	}

	// test (cheap) general case first
	if length <= defaultBufSize || length <= cap(b.buf) {
		return b.buf[:length]
	}

	if length < maxPacketSize {
		b.buf = make([]byte, length)
		return b.buf
	}
	return make([]byte, length)
}

// shortcut which can be used if the requested buffer is guaranteed to be
// smaller than defaultBufSize
// Only one buffer (total) can be used at a time.
func (b *buffer) takeSmallBuffer(length int) []byte {
	if b.length == 0 {
		return b.buf[:length]
	}
	return nil
}

// takeCompleteBuffer returns the complete existing buffer.
// This can be used if the necessary buffer size is unknown.
// Only one buffer (total) can be used at a time.
func (b *buffer) takeCompleteBuffer() []byte {
	if b.length == 0 {
		return b.buf
	}
	return nil
}
