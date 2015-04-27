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
	"strconv"
)

var (
	ErrorScamperBin = errors.New("scamper file is not an executable")
)

const (
	IPv4       = "-4"
	IPv6       = "-6"
	PORT       = "-P"
	SOCKET_DIR = "-U"
	SUDO       = "/usr/bin/sudo"
)

type ScamperConfig struct {
	Port   string
	Path   string
	ScPath string
}

type scamperTool struct {
	sockDir string
}

func ParseScamperConfig(sc ScamperConfig) error {
	val, err := strconv.Atoi(sc.Port)
	if err != nil {
		return err
	}
	if val < 1 || val > 65535 {
		return util.ErrorInvalidPort
	}
	err = checkScamperSockDir(sc.Path)
	if err != nil {
		return err
	}
	return checkScamperBinPath(sc.ScPath)

}

func checkScamperBinPath(binPath string) error {
	fi, err := os.Stat(binPath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return ErrorScamperBin
	}
	return nil
}

func makeScamperDir(sockDir string) error {
	return os.Mkdir(sockDir, os.ModeDir|0700)
}

func checkScamperSockDir(sockDir string) error {
	fi, err := os.Stat(sockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return makeScamperDir(sockDir)
		}
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("Socket directory path: %s is not a directory",
			sockDir)
	}
	return nil
}

func GetProc(sockDir, scampPort, scamperPath string) *proc.Process {

	err := checkScamperSockDir(sockDir)
	if err != nil {
		glog.Fatal("Error with scamper socket directory: %v", err)
	}
	return proc.New(SUDO, nil, scamperPath,
		IPv4, PORT, scampPort, SOCKET_DIR, sockDir)
}

func GetMeasurementTool(sockDir string) *scamperTool {
	return nil
}

func (st *scamperTool) TraceRoute() {

}

func (st *scamperTool) Ping() {

}
