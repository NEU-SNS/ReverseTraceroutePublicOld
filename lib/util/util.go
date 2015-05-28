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
	"fmt"
	"github.com/golang/glog"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, nil, err
	}
	//shortcut, maybe resolve?
	if ip == "localhost" {
		ip = "127.0.0.1"
	}
	pport, err := strconv.Atoi(port)
	if err != nil {
		glog.Errorf("Failed to parse port")
		return 0, nil, err
	}
	if pport < 1 || pport > 65535 {
		glog.Errorf("Invalid port passed to Start: %d", pport)
		return 0, nil, ErrorInvalidPort
	}
	var pip net.IP
	var cont bool
	if ip == "" {
		pip = nil
		cont = true
	} else {
		pip = net.ParseIP(ip)
	}
	if pip == nil && !cont {
		glog.Errorf("Invalid IP passed to Start: %s", ip)
		return 0, nil, ErrorInvalidIP
	}
	return pport, pip, nil
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

func ConvertBytes(path string, b []byte) ([]byte, error) {
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return []byte{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []byte{}, err
	}
	err = cmd.Start()
	if err != nil {
		return []byte{}, err
	}
	_, err = stdin.Write(b)
	if err != nil {
		return []byte{}, err
	}
	err = stdin.Close()
	if err != nil {
		return []byte{}, err
	}
	res := make([]byte, 1024*5)
	_, err = stdout.Read(res)
	if err != nil {
		return res, err
	}
	err = cmd.Wait()
	if err != nil {
		return []byte{}, err
	}
	return res, err
}

func StartPProf(addr string) {
	go func() {
		log.Println(http.ListenAndServe(addr, nil))
	}()
}

func IpStringToInt64(ips string) (int64, error) {
	ip := net.ParseIP(ips)
	if ip == nil {
		return 0, fmt.Errorf("Nil ip in IpToInt64")
	}
	ip = ip.To4()
	var res int64
	res |= int64(ip[0]) << 24
	res |= int64(ip[1]) << 16
	res |= int64(ip[2]) << 8
	res |= int64(ip[3])
	return res, nil
}

func Int64ToIpString(ip int64) (string, error) {
	var a, b, c, d byte
	d = byte(ip & 0x70000000000000ff)
	c = byte(ip & 0x700000000000ff00 >> 8)
	b = byte(ip & 0x7000000000ff0000 >> 16)
	a = byte(ip & 0x70000000ff000000 >> 24)
	nip := net.IPv4(a, b, c, d)
	if nip == nil {
		return "", fmt.Errorf("Invalid IP")
	}
	return nip.String(), nil
}

func MicroToNanoSec(usec int64) int64 {
	return usec * 1000
}
