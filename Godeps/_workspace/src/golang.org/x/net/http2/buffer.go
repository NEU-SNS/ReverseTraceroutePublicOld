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
// Copyright 2014 The Go Authors.
// See https://code.google.com/p/go/source/browse/CONTRIBUTORS
// Licensed under the same terms as Go itself:
// https://code.google.com/p/go/source/browse/LICENSE

package http2

import (
	"errors"
)

// buffer is an io.ReadWriteCloser backed by a fixed size buffer.
// It never allocates, but moves old data as new data is written.
type buffer struct {
	buf    []byte
	r, w   int
	closed bool
	err    error // err to return to reader
}

var (
	errReadEmpty   = errors.New("read from empty buffer")
	errWriteClosed = errors.New("write on closed buffer")
	errWriteFull   = errors.New("write on full buffer")
)

// Read copies bytes from the buffer into p.
// It is an error to read when no data is available.
func (b *buffer) Read(p []byte) (n int, err error) {
	n = copy(p, b.buf[b.r:b.w])
	b.r += n
	if b.closed && b.r == b.w {
		err = b.err
	} else if b.r == b.w && n == 0 {
		err = errReadEmpty
	}
	return n, err
}

// Len returns the number of bytes of the unread portion of the buffer.
func (b *buffer) Len() int {
	return b.w - b.r
}

// Write copies bytes from p into the buffer.
// It is an error to write more data than the buffer can hold.
func (b *buffer) Write(p []byte) (n int, err error) {
	if b.closed {
		return 0, errWriteClosed
	}

	// Slide existing data to beginning.
	if b.r > 0 && len(p) > len(b.buf)-b.w {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	// Write new data.
	n = copy(b.buf[b.w:], p)
	b.w += n
	if n < len(p) {
		err = errWriteFull
	}
	return n, err
}

// Close marks the buffer as closed. Future calls to Write will
// return an error. Future calls to Read, once the buffer is
// empty, will return err.
func (b *buffer) Close(err error) {
	if !b.closed {
		b.closed = true
		b.err = err
	}
}
