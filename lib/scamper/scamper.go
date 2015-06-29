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
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
)

// Options for the scamper process
const (
	// IPv4 Mode
	IPv4 = "-4"
	// IPv6 Mode
	IPv6 = "-6"
	// PORT to use
	PORT = "-P"
	// SOCKETDIR The Socket directory
	SOCKETDIR = "-U"
	// REMOTE Flag
	REMOTE = "-R"
	// SUDO the cmd must be run with sudo
	SUDO      = "/usr/bin/sudo"
	ADDRINDEX = 2
)

type cmdMap struct {
	sync.Mutex
	cmds map[uint32]*Cmd
}

var (
	// ErrorCmdNotFound returned when no cmd is found in the cmdMap
	ErrorCmdNotFound = fmt.Errorf("No command found matching given Id")
	// ErrorDupCommand returned when a socket as a cmd with the same id already
	// running
	ErrorDupCommand = fmt.Errorf("Command already exists with the give Id")
)

func (cm *cmdMap) getCmd(id uint32) (*Cmd, error) {
	cm.Lock()
	defer cm.Unlock()
	if cmd, ok := cm.cmds[id]; ok {
		return cmd, nil
	}
	return nil, ErrorCmdNotFound
}

func (cm *cmdMap) rmCmd(id uint32) {
	cm.Lock()
	defer cm.Unlock()
	delete(cm.cmds, id)
}

func (cm *cmdMap) addCmd(c *Cmd) error {
	cm.Lock()
	defer cm.Unlock()
	if _, ok := cm.cmds[c.userID]; ok {
		return ErrorDupCommand
	}
	cm.cmds[c.userID] = c
	return nil
}

func newCmdMap() *cmdMap {
	m := make(map[uint32]*Cmd)
	return &cmdMap{cmds: m}
}

// Socket represents a scamper control socket
type Socket struct {
	fname     string
	ip        string
	port      string
	closeChan chan struct{}
	errChan   <-chan struct{}
	cmds      *cmdMap
	con       net.Conn
	//Protect userID
	mu     sync.Mutex
	userID uint32
}

func (s *Socket) getID() uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.userID
	s.userID++
	return id
}

func (s *Socket) DoMeasurement(arg interface{}) error {
	cmd, err := newCmd(arg, s.getID())
	if err != nil {
		return err
	}
	err = s.cmds.addCmd(&cmd)
	if err != nil {
		return err
	}
	err = cmd.IssueCommand(s.con)
	return nil
}

// CancelCmd cancels the running scamper command
func (c *Client) CancelCmd() error {
	glog.Info("Canceling command: %d", c.id)
	err := c.checkConn()
	if err != nil {
		return err
	}
	defer c.closeConnection()
	cstring := fmt.Sprintf(cancelCmd, c.id)
	_, err = c.rw.WriteString(cstring)
	if err != nil {
		return err
	}
	glog.Flush()
	return c.rw.Flush()
}

// IssueCmd runs the command on scamper
func (c *Client) IssueCmd(ec chan error, dc chan struct{}) {
	glog.Infof("Issuing command: %s", c.cmd.String())
	err := c.checkConn()
	if err != nil {
		ec <- err
		return
	}
	defer c.closeConnection()
	_, err = c.rw.WriteString(c.cmd.String())
	if err != nil {
		ec <- err
		return
	}
	c.rw.Flush()
	i := 0
	for i < 3 {
		line, err := c.rw.ReadString('\n')
		if err != nil {
			ec <- err
			return
		}
		r, err := parseResponse(line, c.rw)
		if err != nil {
			glog.Errorf("Error parsing response: %v", err)
			ec <- err
			return
		}
		switch r.rType {
		case OK:
			c.id = r.id
		case DATA:
			glog.Infof("Parsed data response")
			c.resps = append(c.resps, r)
			i++
			glog.Infof("Count of data received: %d", i)
		case ERR:
			glog.Errorf("Parsed scamper ERR return")
			ec <- fmt.Errorf("Error with scamper request: %s", c.cmd.String())
			return
		case MORE:
		}

	}
	close(dc)
	return
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

// NewSocket creates a new scamper socket
func NewSocket(fname string) *Socket {
	con, err := net.Dial("unix", fname)
	if err != nil {
		return nil
	}
	sock := &Socket{fname: fname, cmds: newCmdMap(), con: con}
	go func() {
		if err := sock.resetconnection(false); err != nil {
			glog.Errorf("Couldn't start connecton on socket: %s", sock.fname)
			return
		}
		go sock.monitorConnection()
	}()
	return sock
}

func (s *Socket) monitorConnection() {
	for {
		select {
		case <-s.closeChan:
			s.resetconnection(true)
			return
		case <-s.errChan:
			if err := s.resetconnection(false); err != nil {
				glog.Errorf("Socket: %s could not reconnect: %v", s.fname, err)
				return
			}
		}
	}
}

func (s *Socket) resetconnection(close bool) error {
	for {
		if close {
			return s.con.Close()
		}
		con, err := net.Dial("udp", s.fname)
		if err != nil {

			return nil
		}
		s.con = con
		return
	}
}

// Config is the configuration options for the scamper process
type Config struct {
	Port         string
	Path         string
	ScPath       string
	IP           string
	ScParserPath string
}

// ParseConfig checks the given confiuration options to ensure validity
func ParseConfig(sc Config) error {
	val, err := strconv.Atoi(sc.Port)
	if err != nil {
		return err
	}
	if val < 1 || val > 65535 {
		return util.ErrorInvalidPort
	}
	if sc.Path != "" {
		err = checkScamperSockDir(sc.Path)
		if err != nil {
			return err
		}
	}
	return checkBinPath(sc.ScPath)
}

func checkBinPath(binPath string) error {
	fi, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {

			return fmt.Errorf("scamper path does not exist: %s", binPath)
		}
		return err
	}
	if fi.IsDir() {
		return fmt.Errorf("scamper path is not an executable: %s", binPath)
	}
	return nil
}

func makeScamperDir(sockDir string) error {
	return util.MakeDir(sockDir, os.ModeDir|0700)
}

func checkScamperSockDir(sockDir string) error {
	isd, err := util.IsDir(sockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return makeScamperDir(sockDir)
		}
		return err
	}
	if isd {
		return nil

	}
	return fmt.Errorf("Socket directory path: %s is not a directory",
		sockDir)
}

// GetProc returns a process which will run scamper
func GetProc(sockDir, scampPort, scamperPath string) *proc.Process {

	err := checkScamperSockDir(sockDir)
	if err != nil {
		glog.Errorf("Error with scamper socket directory: %v", err)
		return nil
	}
	return proc.New(scamperPath, nil,
		IPv4, PORT, scampPort, SOCKETDIR, sockDir)
}

// GetVPProc returns a process which is suitable to run on a planet-lab VP
func GetVPProc(scpath, host, port string) *proc.Process {
	faddr := fmt.Sprintf("%s:%s", host, port)
	return proc.New(SUDO, nil, scpath, REMOTE, faddr)
}
