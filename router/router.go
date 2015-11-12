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
	"fmt"
	"sync"

	"golang.org/x/net/context"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

type service uint

const (
	planetLab service = iota + 1
)

var (
	errCantCreateMt = fmt.Errorf("No measurement tool found for the service")
)

// MeasurementTool is the interface for a measurement source the controller can use
type MeasurementTool interface {
	Ping(context.Context, *dm.PingArg) (<-chan *dm.Ping, error)
	Traceroute(context.Context, *dm.TracerouteArg) (<-chan *dm.Traceroute, error)
	GetVPs(context.Context, *dm.VPRequest) (<-chan *dm.VPReturn, error)
	ReceiveSpoof(context.Context, *dm.RecSpoof) (<-chan *dm.NotifyRecSpoofResponse, error)
	Close() error
}

func create(s ServiceDef) (MeasurementTool, error) {
	switch s.Service {
	case planetLab:
		return createPLMT(s)
	}
	return nil, errCantCreateMt
}

// ServiceDef is the definition of
type ServiceDef struct {
	Addr    string
	Port    string
	Service service
}

type source struct{}

func (s source) Get(dst string) (ServiceDef, error) {
	return ServiceDef{
		Addr:    "plcontroller.revtr.ccs.neu.edu",
		Port:    "4380",
		Service: planetLab,
	}, nil
}

func (s source) All() []ServiceDef {
	return []ServiceDef{ServiceDef{
		Addr:    "plcontroller.revtr.ccs.neu.edu",
		Port:    "4380",
		Service: planetLab,
	}}
}

// Source is a source of service defs from src addresses
type Source interface {
	Get(string) (ServiceDef, error)
	All() []ServiceDef
}

type mtCache struct {
	mu    sync.Mutex
	cache map[ServiceDef]*mtCacheItem
}

type mtCacheItem struct {
	mt       MeasurementTool
	refCount uint32
}

// Router is the interface for something that routes srcs to measurement tools
type Router interface {
	GetMT(ServiceDef) (MeasurementTool, error)
	PutMT(ServiceDef)
	GetService(string) (ServiceDef, error)
	All() []MeasurementTool
	SetSource(Source)
}

type router struct {
	source Source
	cache  mtCache
}

// New creates a new Router
func New() Router {
	return &router{
		cache: mtCache{
			cache: make(map[ServiceDef]*mtCacheItem),
		},
		source: source{},
	}
}

func (r *router) SetSource(s Source) {
	r.source = s
}

func (r *router) GetMT(s ServiceDef) (MeasurementTool, error) {
	r.cache.mu.Lock()
	defer r.cache.mu.Unlock()
	if mt, ok := r.cache.cache[s]; ok {
		mt.refCount++
		return mt.mt, nil
	}
	mt, err := create(s)
	if err != nil {
		log.Error(err, s)
		return nil, err
	}
	nc := &mtCacheItem{
		mt:       mt,
		refCount: 1,
	}
	r.cache.cache[s] = nc
	return mt, nil
}

func (r *router) PutMT(s ServiceDef) {
	r.cache.mu.Lock()
	defer r.cache.mu.Unlock()
	if mt, ok := r.cache.cache[s]; ok {
		mt.refCount--
		if mt.refCount == 0 {
			delete(r.cache.cache, s)
		}
		return
	}
	panic("PutMT without GetMT first")
}

func (r *router) GetService(addr string) (ServiceDef, error) {
	return r.source.Get(addr)
}

func (r *router) All() []MeasurementTool {
	services := r.source.All()
	ret := make([]MeasurementTool, len(services))
	for i, s := range services {
		mt, _ := create(s)
		ret[i] = mt
	}
	return ret
}
