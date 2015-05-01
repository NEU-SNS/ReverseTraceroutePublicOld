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
package util

import (
	"bufio"
	"errors"
	"github.com/golang/glog"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"strings"
)

const (
	IP   = 0
	PORT = 1
)

var (
	ErrorInvalidIP   = errors.New("invalid IP address")
	ErrorInvalidPort = errors.New("invalid port")
)

func IsDir(dir string) (bool, error) {
	fi, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

func MakeDir(path string, mode os.FileMode) error {
	return os.Mkdir(path, mode)
}

func ParseAddrArg(addr string) (int, net.IP, error) {
	parts := strings.Split(addr, ":")
	ip := parts[IP]

	//shortcut, maybe resolve?
	if ip == "localhost" {
		ip = "127.0.0.1"
	}
	port := parts[PORT]
	pport, err := strconv.Atoi(port)
	if err != nil {
		glog.Errorf("Failed to parse port")
		return 0, nil, err
	}
	if pport < 1 || pport > 65535 {
		glog.Errorf("Invalid port passed to Start: %d", pport)
		return 0, nil, ErrorInvalidPort
	}
	pip := net.ParseIP(ip)
	if pip == nil {
		glog.Errorf("Invalid IP passed to Start: %s", ip)
		return 0, nil, ErrorInvalidIP
	}
	return pport, pip, nil
}

func StartRpc(n, laddr string, eChan chan error, api interface{}) {
	server := rpc.NewServer()
	server.Register(api)
	l, e := net.Listen(n, laddr)
	if e != nil {
		glog.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	glog.Infof("Controller started, listening on: %s", laddr)
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorf("Accept failed: %v", err)
			eChan <- err
			continue
		}
		glog.Info("Serving reqeust")
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func CloseStdFiles(c bool) {
	if !c {
		return
	}
	glog.Info("Closing standard file descripters")
	defer glog.Flush()
	err := os.Stdin.Close()

	if err != nil {
		glog.Error("Failed to close Stdin")
		os.Exit(1)
	}
	err = os.Stderr.Close()
	if err != nil {
		glog.Error("Failed to close Stderr")
		os.Exit(1)
	}
	err = os.Stdout.Close()
	if err != nil {
		glog.Error("Failed to close Stdout")
		os.Exit(1)
	}
}

func ConnToRW(c net.Conn) *bufio.ReadWriter {
	w := bufio.NewWriter(c)
	r := bufio.NewReader(c)
	rw := bufio.NewReadWriter(r, w)
	return rw
}
