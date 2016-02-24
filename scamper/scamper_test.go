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

package scamper_test

import (
	"net"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var sockPath = "/tmp/192.168.1.2:5000"
var testIP = "192.168.1.2"
var testPort = "5000"

func setupSocket(t *testing.T) func() {
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to setupSocket: %v", err)
	}
	donec := make(chan struct{})
	go func() {
		for {
			select {
			case <-donec:
				l.Close()
				return
			default:
				l.Accept()
			}
		}
	}()
	return func() {
		close(donec)
	}
}

func TestSocketStop(t *testing.T) {
	done := setupSocket(t)
	defer util.LeakCheck(t)()
	sock, err := scamper.NewSocket(sockPath, net.Dial)
	if err != nil {
		t.Fatalf("Failed to create a socket: %v", err)
	}
	sock.Stop()
	done()
}

func TestSocketIP_Port(t *testing.T) {
	done := setupSocket(t)
	defer util.LeakCheck(t)()
	sock, err := scamper.NewSocket(sockPath, net.Dial)
	if err != nil {
		t.Fatalf("Failed to create a socket: %v", err)
	}
	if sock.IP() != "192.168.1.2" {
		t.Fatalf("Failed getting socket IP expected[192.168.1.2] got[%s]", sock.IP())
	}
	if sock.Port() != "5000" {
		t.Fatalf("Failed getting socket Port expected[5000] got [%s]", sock.Port())
	}
	sock.Stop()
	done()
}
