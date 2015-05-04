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
	"errors"
	"fmt"
	"github.com/NEU-SNS/ReverseTraceroute/lib/mproc/proc"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	ErrorScamperBin = errors.New("scamper file is not an executable")
)

const (
	IPv4       = "-4"
	IPv6       = "-6"
	PORT       = "-P"
	SOCKET_DIR = "-U"
	REMOTE     = "-R"
	SUDO       = "/usr/bin/sudo"
)

type Socket struct {
	fname string
	ip    string
	port  string
}

func (s Socket) IP() string {
	if s.ip == "" {
		s.ip = strings.Split(path.Base(s.fname), ":")[util.IP]
		return s.ip
	}
	return s.ip
}

func (s Socket) Port() string {
	if s.port == "" {
		s.port = strings.Split(path.Base(s.fname), ":")[util.PORT]
		return s.port
	}
	return s.port
}

func NewSocket(fname string) Socket {
	return Socket{fname: fname}
}

type ScamperConfig struct {
	Port   string
	Path   string
	ScPath string
	Url    string
}

func ParseScamperConfig(sc ScamperConfig) error {
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

	return checkScamperBinPath(sc.ScPath)

}

func checkScamperBinPath(binPath string) error {
	fi, err := os.Stat(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrorScamperBin
		}
		return err
	}
	if fi.IsDir() {
		return ErrorScamperBin
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

func GetProc(sockDir, scampPort, scamperPath string) *proc.Process {

	err := checkScamperSockDir(sockDir)
	if err != nil {
		glog.Errorf("Error with scamper socket directory: %v", err)
		return nil
	}
	return proc.New(SUDO, nil, scamperPath,
		IPv4, PORT, scampPort, SOCKET_DIR, sockDir)
}

func GetVPProc(scpath, host, port string) *proc.Process {
	faddr := fmt.Sprintf("%s:%s", host, port)
	return proc.New(SUDO, nil, scpath, REMOTE, faddr)
}
