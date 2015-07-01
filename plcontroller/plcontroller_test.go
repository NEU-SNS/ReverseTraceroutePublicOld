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
package plcontroller

import (
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/scamper"
)

var conf = Config{Local: LocalConfig{Addr: "localhost:45000",
	Proto: "tcp"}, Scamper: ScamperConfig{Port: "45454", SockDir: "/tmp/scamper_sockets", BinPath: "/usr/local/bin/sc_remoted",
	ConverterPath: "/usr/local/bin/sc_warts2json"}}

func TestStart(t *testing.T) {
	eChan := Start(conf, true)

	select {
	case e := <-eChan:
		t.Fatal("TestStart failed %v", e)
	case <-time.After(time.Second * 2):

	}

}

func TestStartInvalidIP(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "-1:45000",
		Proto: "tcp"}, Scamper: ScamperConfig{Port: "45454", SockDir: "/tmp/scamper_sockets", BinPath: "/usr/local/bin/sc_remoted",
		ConverterPath: "/usr/local/bin/sc_warts2json"}}
	eChan := Start(c, true)
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartInvalidIP no error thrown with invalid ip")
	}

}

func TestStartInvalidPort(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "localhost:PORT",
		Proto: "tcp"}, Scamper: ScamperConfig{Port: "45454", SockDir: "/tmp/scamper_sockets", BinPath: "/usr/local/bin/sc_remoted",
		ConverterPath: "/usr/local/bin/sc_warts2json"}}
	eChan := Start(c, true)
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartInvalidPort no error thrown with invalid port")
	}

}

func TestStartPortOutOfRange(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "localhost:70000",
		Proto: "tcp"}, Scamper: ScamperConfig{Port: "45454", SockDir: "/tmp/scamper_sockets", BinPath: "/usr/local/bin/sc_remoted",
		ConverterPath: "/usr/local/bin/sc_warts2json"}}
	eChan := Start(c, true)
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartPortOutOfRange no error thrown with port 70000")
	}
}

func TestGetStats(t *testing.T) {
	stat := plController.getStats()
	if stat.TotReqTime != 0 && stat.Requests != 0 {
		t.Fatal("Stats returned incorrect data")
	}
}

func TestGetStatsInfo(t *testing.T) {
	tt, c := plController.getStatsInfo()
	if tt != 0 && c != 0 {
		t.Fatal("Stats returned incorrect data")
	}
}

func TestIncreaseStats(t *testing.T) {
	tim := time.Now()
	tt, c := plController.getStatsInfo()
	plController.increaseStats(tim)
	ft, fc := plController.getStatsInfo()
	if ft <= tt || fc <= c {
		t.Fatal("Increase stats failed to increase stats")
	}
}

func TestAddSocket(t *testing.T) {
	path := "/tmp/fake"
	getName := "fake"
	sock := scamper.NewSocket(path)
	plController.addSocket(sock)
	sock, err := plController.getSocket(getName)
	if err != nil {
		t.Fatal("Failed to add socket")
	}
}

func TestRemoveSocket(t *testing.T) {
	path := "/tmp/fake"
	getName := "fake"
	sock := scamper.NewSocket(path)
	plController.addSocket(sock)
	plController.removeSocket(sock)
	sock, err := plController.getSocket(getName)
	if err == nil {
		t.Fatal("Failed to remove socket")
	}
}
