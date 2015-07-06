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

// Package controller is the library for creating a central controller
package controller

import (
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/cache"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess/testdataaccess"
)

var conf = Config{Local: LocalConfig{Addr: "localhost:45000",
	Proto: "tcp"}}

func TestStart(t *testing.T) {
	eChan := Start(conf, da.New(), cache.New())

	select {
	case e := <-eChan:
		t.Fatalf("TestStart failed %v", e)
	case <-time.After(time.Second * 2):

	}

}

func TestStartNoDB(t *testing.T) {
	eChan := Start(conf, nil, cache.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("Controller started with nil DB")
	}

}

func TestStartInvalidIP(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "-1:45000",
		Proto: "tcp"}}
	eChan := Start(c, da.New(), cache.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatalf("TestStartInvalidIP no error thrown with invalid ip")
	}

}

func TestStartInvalidPort(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "127.0.0.1:PORT",
		Proto: "tcp"}}
	eChan := Start(c, da.New(), cache.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatalf("TestStartInvalidPort no error thrown with invalid port")
	}

}

func TestStartPortOutOfRange(t *testing.T) {
	var c = Config{Local: LocalConfig{Addr: "127.0.0.1:70000",
		Proto: "tcp"}}
	eChan := Start(c, da.New(), cache.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatalf("TestStartPortOutOfRange no error thrown with port 70000")
	}
}

func TestGetRequests(t *testing.T) {
	r := controller.getRequests()
	if r < 0 {
		t.Fatalf("Invalid requests num returned: %d", r)
	}
}

func TestAddRequest(t *testing.T) {
	s := controller.getRequests()
	controller.addRequest()
	f := controller.getRequests()
	if f != s+1 {
		t.Fatalf("Add request failed, got %d expected %d", f, s+1)
	}
}

func TestAddTime(t *testing.T) {
	s := controller.getTime()
	controller.addTime(2 * time.Second)
	f := controller.getTime()
	if f != s+(2*time.Second) {
		t.Fatalf("Add time failed, got %d expected %d", f, s+2*time.Second)
	}
}

func TestAddReqStats(t *testing.T) {
	stat := controller.getStats()
	req := Request{Dur: 2 * time.Second}
	controller.addReqStats(req)
	fstat := controller.getStats()
	if fstat.Requests != stat.Requests+1 ||
		time.Duration(fstat.TotReqTime) != time.Duration(stat.TotReqTime)+(2*time.Second) {
		t.Fatalf("Add req Stats failed, got %v expected %v", fstat, stat)
	}
}
