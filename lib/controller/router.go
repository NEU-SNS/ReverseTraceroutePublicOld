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
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
	"sync"
)

type router struct {
	rw          sync.RWMutex
	services    map[dm.ServiceT]*dm.Service
	servClients map[dm.ServiceT]MeasurementTool
}

func createRouter() Router {
	s := make(map[dm.ServiceT]*dm.Service)
	sc := make(map[dm.ServiceT]MeasurementTool)
	r := &router{services: s, servClients: sc}
	r.registerClients()
	return r
}

func (r *router) registerClients() {
	r.servClients[dm.ServiceT_PLANET_LAB] = &plClient{}
}

func NewRouter() Router {
	return createRouter()
}

type Router interface {
	RegisterServices(services ...*dm.Service)
	GetService(dm.ServiceT) (*dm.Service, MeasurementTool, error)
	GetServices() []*dm.Service
}

func (r *router) GetServices() []*dm.Service {
	r.rw.RLock()
	serv := make([]*dm.Service, len(r.services), len(r.services))
	for _, service := range r.services {
		serv = append(serv, service)
	}
	r.rw.RUnlock()
	return serv
}

func (r *router) RegisterServices(services ...*dm.Service) {
	r.rw.Lock()
	for _, service := range services {
		r.services[service.Key] = service
		glog.Infof("Registered service: %v", service)
	}
	r.rw.Unlock()
}

func (r *router) GetService(s dm.ServiceT) (*dm.Service, MeasurementTool, error) {
	r.rw.RLock()
	glog.Infof("Trying to get API for %s", s)
	serv := r.services[s]
	if serv == nil {
		return nil, nil, ErrorServiceNotFound
	}
	mt := r.servClients[s]
	if mt == nil {
		return nil, nil, ErrorServiceNotFound
	}
	r.rw.RUnlock()
	return serv, mt, nil
}
