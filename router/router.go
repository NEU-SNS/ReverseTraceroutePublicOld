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
package router

import (
	"golang.org/x/net/context"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type Service uint

const (
	PLANET_LAB Service = iota + 1
)

type MeasurementTool interface {
	Ping(context.Context, *dm.PingArg) (<-chan *dm.Ping, error)
	Traceroute(context.Context, *dm.TracerouteArg) (<-chan *dm.Traceroute, error)
	GetVPs(context.Context, *dm.VPRequest) (<-chan *dm.VPReturn, error)
}

func create(s ServiceDef) (MeasurementTool, error) {
	switch s.Service {
	case PLANET_LAB:
		return nil, nil
	}
	return nil, nil
}

type ServiceDef struct {
	Addr    string
	Port    string
	Service Service
}

type Source interface {
	Get(string) (ServiceDef, error)
	All() []ServiceDef
}

type Router interface {
	Get(string) (MeasurementTool, error)
	All() []MeasurementTool
	SetSource(Source)
}

type mtCache map[ServiceDef]MeasurementTool

type router struct {
	source      Source
	activeCache mtCache
}

func New() Router {
	return &router{
		activeCache: make(mtCache),
	}
}

func (r *router) SetSource(s Source) {
	r.source = s
}

func (r *router) Get(addr string) (MeasurementTool, error) {
	service, err := r.source.Get(addr)
	if err != nil {
		return nil, err
	}
	return create(service)
}

func (r *router) All() []MeasurementTool {
	return nil
}
