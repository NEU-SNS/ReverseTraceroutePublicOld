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

// Package scamper is a library to work with scamper control sockets
package scamper

import (
	"errors"
	"io"
	"sync"

	"github.com/golang/glog"
)

// SResponseT represents the type of responses scamper can send
type SResponseT string

const (
	// OK is the accept response from scamper
	OK SResponseT = "OK"
	// MORE is the response when more commands can be given
	MORE SResponseT = "MORE"
	// DATA represensts a data message
	DATA SResponseT = "DATA"
	// ERR is the error response from scamper
	ERR SResponseT = "ERR"
)

var (
	// ErrorBadDataResponse is returned when the data received by scamper couldnt
	// be converted
	ErrorBadDataResponse = errors.New("Bad DATA Response")
	// ErrorBadOKResponse is returned when an OK response fails to parse
	ErrorBadOKResponse = errors.New("Bad OK Response")
	// ErrorBadResponse is the generic error when a response can't be parsed
	ErrorBadResponse = errors.New("Bad Response")
	// ErrorTimeout returned when a command times out
	ErrorTimeout = errors.New("Timeout")
)

//TODO: A possible optimization to make is to open a single connection,
// resuse it until it fails, and using the -U option to assign an id to
// each measurement and then return them to the proper caller. This
// would also involve blocking on a conn waiting for data at any time
// measurements are out.

// Response represents a response from scamper
type Response struct {
	rType  SResponseT
	data   []byte
	ds     int
	userID int
	ret    interface{}
}

type socketMap struct {
	sync.Mutex
	socks map[string]*Socket
}

func newSocketMap() *socketMap {
	s := make(map[string]*Socket)
	return &socketMap{socks: s}
}

func (sm *socketMap) add(s *Socket) {
	sm.Lock()
	defer sm.Unlock()
	sm.socks[s.IP()] = s
}

func (sm *socketMap) remove(addr string) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.socks, addr)
}

// Client is the main object for interacting with scamper
type Client struct {
	sockets *socketMap
}

// Bytes get the data as bytes from a scamper response
func (r Response) Bytes() []byte {
	return r.data
}

// WriteTo writes the response to the given io.Writer
func (r Response) WriteTo(w io.Writer) (n int64, err error) {
	glog.Infof("Writing data %v", r.data)
	c, err := w.Write(r.data)
	n = int64(c)
	glog.Infof("Wrote %d bytes", n)
	return
}

// NewClient creates a new Client
func NewClient() *Client {
	return &Client{sockets: newSocketMap()}
}
