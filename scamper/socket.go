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
	"bufio"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"sync"

	"sync/atomic"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/uuencode"
	"github.com/NEU-SNS/ReverseTraceroute/warts"
)

var (
	// ErrorCmdNotFound returned when no cmd is found in the cmdMap
	ErrorCmdNotFound = fmt.Errorf("No command found matching given Id")
	// ErrorDupCommand returned when a socket as a cmd with the same id already
	// running
	ErrorDupCommand = fmt.Errorf("Command already exists with the give Id")
)

type cmdMap struct {
	sync.Mutex
	cmds map[uint32]cmdResponse
}

type cmdResponse struct {
	cmd  *Cmd
	done chan Response
}

func (cm *cmdMap) forEach() <-chan cmdResponse {
	c := make(chan cmdResponse)
	go func() {
		for key, sock := range cm.cmds {
			delete(cm.cmds, key)
			c <- sock
		}
		close(c)
	}()
	return c
}

func (cm *cmdMap) getCmd(id uint32) (cmdResponse, error) {
	cm.Lock()
	defer cm.Unlock()
	if cmd, ok := cm.cmds[id]; ok {
		return cmd, nil
	}
	return cmdResponse{}, ErrorCmdNotFound
}

func (cm *cmdMap) rmCmd(id uint32) {
	cm.Lock()
	defer cm.Unlock()
	delete(cm.cmds, id)
}

func (cm *cmdMap) addCmd(c cmdResponse) error {
	cm.Lock()
	defer cm.Unlock()
	if _, ok := cm.cmds[c.cmd.userID]; ok {
		return ErrorDupCommand
	}
	cm.cmds[c.cmd.userID] = c
	return nil
}

func newCmdMap() *cmdMap {
	m := make(map[uint32]cmdResponse)
	return &cmdMap{cmds: m}
}

// Socket represents a scamper control socket
type Socket struct {
	fname       string
	ip          string
	port        string
	closeChan   chan struct{}
	errChan     chan error
	cmdChan     chan *Cmd
	respChan    chan Response
	cmds        *cmdMap
	con         io.ReadWriteCloser
	wartsHeader [2]Response
	rc          uint32
	rw          *bufio.ReadWriter
	// Access atomically
	userID uint32
}

// NewSocket creates a new scamper socket
func NewSocket(fname string, con io.ReadWriteCloser) (*Socket, error) {
	cc := make(chan *Cmd, 10)
	rc := make(chan Response, 10)
	clc := make(chan struct{})
	sock := &Socket{
		fname:     fname,
		cmds:      newCmdMap(),
		cmdChan:   cc,
		respChan:  rc,
		closeChan: clc,
		con:       con,
		rw:        bufio.NewReadWriter(bufio.NewReader(con), bufio.NewWriter(con)),
	}

	go sock.monitorConn()
	go sock.readConn()
	return sock, nil
}

// Stop closes the connection the socket represents
func (s *Socket) Stop() {
	if s == nil {
		return
	}
	for cmd := range s.cmds.forEach() {
		close(cmd.done)
	}
	s.con.Close()
	select {
	case <-s.closeChan:
		return
	default:
		close(s.closeChan)
	}
}

func (s *Socket) readConn() {
	for {
		line, err := s.rw.ReadString('\n')
		if err != nil {
			log.Error(err)
			return
		}
		resp, err := parseResponse(line, s.rw)
		if err != nil {
			log.Errorf("Error parsing response: %s", line)
			continue
		}
		if resp.RType != data {
			continue
		}
		// The first two data messages received are the header of the warts format
		if s.rc < 2 {
			s.wartsHeader[s.rc] = resp
			s.rc++
			continue
		}
		dec := &uuencode.UUDecodingWriter{}
		s.wartsHeader[0].WriteTo(dec)
		s.wartsHeader[1].WriteTo(dec)
		resp.WriteTo(dec)
		go func() {
			var filter []warts.WartsT
			filter = append(filter, warts.PingT, warts.TracerouteT)
			res, err := warts.Parse(dec.Bytes(), filter)
			resp.Err = err
			if err != nil {
				log.Errorf("Could not parse response: %s", err)
			}
			if len(res) != 1 {
				log.Errorf("Wrong number of objects parsed from warts, expected 1, got %d", len(res))
			} else {
				switch t := res[0].(type) {
				case warts.Traceroute:
					resp.UserID = t.Flags.UserID
				case warts.Ping:
					resp.UserID = t.Flags.UserID
				}
				resp.Ret = res[0]
				cmdmap, err := s.cmds.getCmd(resp.UserID)
				if err != nil {
					log.Warnf("Failed to get command for id: %d, err: %s", resp.UserID, err)
					return
				}
				s.cmds.rmCmd(resp.UserID)

				select {
				case <-s.closeChan:
				case cmdmap.done <- resp:
				}
			}
		}()
	}

}

func (s *Socket) monitorConn() {
	for {
		select {
		case <-s.closeChan:
			return
		case c := <-s.cmdChan:
			err := c.issueCommand(s.con)
			if err != nil {
				log.Errorf("Error issuing command %s", c.Marshal())
				continue
			}

		}
	}
}

func (s *Socket) getID() uint32 {
	return atomic.AddUint32(&s.userID, 1)
}

// RemoveMeasurement remove a measurment being run with id id
func (s *Socket) RemoveMeasurement(id uint32) error {
	s.cmds.rmCmd(id)
	return nil
}

// DoMeasurement perform the measurement described by arg
func (s *Socket) DoMeasurement(arg interface{}) (<-chan Response, uint32, error) {
	id := s.getID()
	cmd, err := newCmd(arg, id)
	if err != nil {
		return nil, 0, err
	}
	cr := cmdResponse{cmd: &cmd, done: make(chan Response, 1)}
	err = s.cmds.addCmd(cr)
	if err != nil {
		return nil, 0, err
	}
	select {
	case <-s.closeChan:
		s.cmds.rmCmd(id)
		return nil, 0, fmt.Errorf("Socket closed before command could run.")
	case s.cmdChan <- &cmd:
	}
	return cr.done, id, err
}

// IP Gets the ip of the remote machine that is connected to the socket
func (s *Socket) IP() string {
	if s.ip == "" {
		s.ip = strings.Split(path.Base(s.fname), ":")[util.IP]
		return s.ip
	}
	return s.ip
}

// Port gets the port of the remote machine that is connected to the socket
func (s *Socket) Port() string {
	if s.port == "" {
		s.port = strings.Split(path.Base(s.fname), ":")[util.PORT]
		return s.port
	}
	return s.port
}

func parseResponse(r string, rw *bufio.ReadWriter) (Response, error) {
	resp := Response{}
	switch {
	case strings.Contains(r, string(ok)):
		resp.RType = ok
		r = strings.TrimSpace(r)
		split := strings.Split(r, " ")
		idsp := strings.Split(split[1], "-")
		_, err := strconv.Atoi(idsp[1])
		if err != nil {
			return resp, ErrorBadResponse
		}
		return resp, nil
	case strings.Contains(r, string(err)):
		resp.RType = err
		return resp, nil
	case strings.Contains(r, string(data)):
		resp.RType = data
		split := strings.Split(r, " ")
		if len(split) != 2 {
			return resp, ErrorBadDataResponse
		}
		n, err := strconv.Atoi(split[1][:len(split[1])-1])
		if err != nil {
			return resp, err
		}
		resp.DS = n
		buff := make([]byte, n)
		_, err = io.ReadFull(rw, buff)
		if err != nil {

			return resp, err
		}
		resp.Data = buff
		return resp, nil
	case strings.Contains(r, string(more)):
		resp.RType = more
		return resp, nil
	}
	return resp, ErrorBadResponse
}
