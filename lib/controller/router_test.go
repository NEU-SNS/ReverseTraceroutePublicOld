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
     * Neither the name of the University of Washington nor the
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
package controller

import (
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess/testdataaccess"
	"testing"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Errorf("TestNewRouter returned nil router")
	}
}

func TestRegisterServices(t *testing.T) {
	r := NewRouter()
	r.RegisterServices(da.New().GetServices("192.168.1.1")...)
	if len(r.GetServices()) == 0 {
		t.Errorf("TestRegisterServices failed to add service")
	}
}

func TestRouteRequestNoService(t *testing.T) {
	r := NewRouter()
	r.RegisterServices(da.New().GetServices("192.168.1.1")...)
	req := Request{
		Key: "Fake Key",
	}
	_, err := r.RouteRequest(req)
	if err != ErrorServiceNotFound {
		t.Error("TestRouteRequestNoService didnt return ErrorServiceNotFound",
			" with fake service")
	}
}

func TestRouteRequest(t *testing.T) {
	r := NewRouter()
	r.RegisterServices(da.New().GetServices("192.168.1.1")...)
	req := Request{
		Key: "TEST",
	}

	rr, err := r.RouteRequest(req)
	if rr == nil || err != nil {
		t.Errorf("TestRouteRequest failed: %v, %v", rr, err)
	}
}

func TestRouteRequestRunRequest(t *testing.T) {
	r := NewRouter()
	r.RegisterServices(da.New().GetServices("192.168.1.1")...)
	req := Request{
		Key: "TEST",
	}

	rr, err := r.RouteRequest(req)
	result, _, err := rr()
	if result == nil && err == nil {
		t.Errorf("TestRouteRequestRunRequest failed %v, %v", result, err)
	}
}
