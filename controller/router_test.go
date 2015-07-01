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
package controller

import (
	"testing"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

func TestNewRouter(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Fatalf("TestNewRouter returned nil router")
	}
}

var service = &dm.Service{
	Url:  "fakepl",
	Key:  dm.ServiceT_PLANET_LAB,
	Port: 9999,
}

var servs = []*dm.Service{service}

func TestRegisterServices(t *testing.T) {
	r := NewRouter()
	if r == nil {
		t.Fatalf("TestNewRouter returned nil router")
	}

	r.RegisterServices(servs...)
}

func setupRouter(t *testing.T) Router {
	r := NewRouter()
	if r == nil {
		t.Fatalf("TestNewRouter returned nil router")
	}
	r.RegisterServices(servs...)
	return r
}

func TestGetClient(t *testing.T) {
	r := setupRouter(t)
	s, mt, err := r.GetClient(dm.ServiceT_PLANET_LAB)
	if s == nil || mt == nil || err != nil {
		t.Fatalf("TestGetClient Failed: %v, %v, %v", s, mt, err)
	}
}

func TestGetClientUnknownService(t *testing.T) {
	r := setupRouter(t)
	s, mt, err := r.GetClient(dm.ServiceT(-1))
	if s != nil || mt != nil || err != nil {
		t.Fatalf("TestGetClientUnknowService Failed: %v, %v, %v", s, mt, err)
	}
}

func TestGetService(t *testing.T) {
	r := setupRouter(t)
	cl, mt, err := r.GetService(dm.ServiceT_PLANET_LAB)
	if cl == nil || mt == nil || err != nil {
		t.Fatalf("TestGetService Failed: %v, %v, %v", cl, mt, err)
	}
}

func TestGetServiceUnknownService(t *testing.T) {
	r := setupRouter(t)
	cl, mt, err := r.GetService(dm.ServiceT(-1))
	if cl != nil || err == nil {
		t.Fatalf("TestGetServiceUnknowService Failed: %v, %v, %v", cl, mt, err)
	}
}

func TestGetServiceBadMT(t *testing.T) {
	r := setupRouter(t)
	rout := r.(*router)
	delete((*rout).servClients, dm.ServiceT_PLANET_LAB)
	cl, mt, err := rout.GetService(dm.ServiceT_PLANET_LAB)
	if err == nil || err != ErrorServiceNotFound {
		t.Fatalf("TestGetServiceUnknowService Failed: %v, %v, %v", cl, mt, err)
	}
}

func TestGetServices(t *testing.T) {
	r := setupRouter(t)
	services := r.GetServices()
	if services == nil ||
		len(services) != 1 ||
		services[0] != service {
		t.Fatalf("TestGetServices Failed: services: %v, got: %v", servs, services[0])
	}
}
