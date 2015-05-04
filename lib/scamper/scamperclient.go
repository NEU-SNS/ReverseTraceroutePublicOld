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
package scamper

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"io"
	"net"
	"strconv"
	"strings"
)

type SResponseT string

const (
	OK   SResponseT = "OK"
	MORE SResponseT = "MORE"
	DATA SResponseT = "DATA"
	ERR  SResponseT = "ERR"
)

var (
	ErrorBadDataResponse = errors.New("Bad DATA Response")
	ErrorBadOKResponse   = errors.New("Bad OK Response")
	ErrorBadResponse     = errors.New("Bad Response")
)

type Response struct {
	rType SResponseT
	data  []byte
	ds    int
}

type Client struct {
	rw    *bufio.ReadWriter
	s     Socket
	cmd   Cmd
	resps []Response
}

func (r Response) WriteTo(w io.Writer) error {
	//Num of bytes that need to be written
	l := len(r.data)
	//Start of where to write from
	s := 0
	for l > 0 {
		//Write bytes
		c, err := w.Write(r.data[s:])
		if err != nil && err != io.ErrShortWrite {
			return err
		}
		l -= c
		s += c
	}
	return nil
}

func NewClient(s Socket, c Cmd) Client {
	return Client{s: s, cmd: c, resps: make([]Response, 1)}
}

func (c *Client) GetResponses() []Response {
	return c.resps
}

func (c *Client) checkConn() error {
	if !c.connected() {
		return c.connect()
	}
	return nil
}

func (c *Client) IssueCmd() error {
	glog.Infof("Issuing command: %s", c.cmd.String())
	err := c.checkConn()
	if err != nil {
		return err
	}
	_, err = c.rw.WriteString(c.cmd.String())
	if err != nil {
		return err
	}
	c.rw.Flush()
	i := 0
	for i < 3 {
		line, err := c.rw.ReadString('\n')
		if err != nil {
			return err
		}
		r, err := parseResponse(line, c.rw)
		if err != nil {
			glog.Errorf("Error parsing response: %v", err)
			return err
		}
		switch {
		case r.rType == OK:
		case r.rType == DATA:
			glog.Infof("Parsed data response")
			c.resps = append(c.resps, r)
			i += 1
			glog.Infof("Count of data received: %d", i)
		case r.rType == ERR:
			glog.Errorf("Parsed scamper ERR return")
			return fmt.Errorf("Error with scamper request: %s", c.cmd.String())
		case r.rType == MORE:
		}

	}
	return nil
}

func (c *Client) connect() error {
	glog.Infof("Connecting to: %s", c.s.fname)
	conn, err := net.Dial("unix", c.s.fname)
	if err != nil {
		return err
	}
	c.rw = util.ConnToRW(conn)
	return nil
}

func (c *Client) connected() bool {
	return c.rw != nil
}

func parseResponse(r string, rw *bufio.ReadWriter) (Response, error) {
	resp := Response{}
	resp.data = make([]byte, 10)
	glog.Infof("Parsing Response")
	switch {
	case strings.Contains(r, string(OK)):
		resp.rType = OK
		return resp, nil
	case strings.Contains(r, string(ERR)):
		glog.Error("Parsed error response")
		resp.rType = ERR
		return resp, nil
	case strings.Contains(r, string(DATA)):
		resp.rType = DATA
		split := strings.Split(r, " ")
		if len(split) != 2 {
			return resp, ErrorBadDataResponse
		}
		n, err := strconv.Atoi(split[1][:len(split[1])-1])
		if err != nil {
			return resp, err
		}
		resp.ds = n
		buff := make([]byte, n)
		_, err = io.ReadFull(rw, buff)
		if err != nil {
			return resp, err
		}
		resp.data = buff
		return resp, nil
	case strings.Contains(r, string(MORE)):
		resp.rType = MORE
		return resp, nil
	}
	return resp, ErrorBadResponse
}
