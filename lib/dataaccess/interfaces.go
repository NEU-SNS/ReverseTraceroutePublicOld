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
package dataaccess

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"time"
)

type DataProvider interface {
	ServiceProvider
	TracerouteProvider
	PingProvider
	Close() error
}

type ServiceProvider interface {
	GetServices() ([]*dm.Service, error)
}

type Staleness time.Duration

type TracerouteProvider interface {
	StoreTraceroute(*dm.Traceroute, dm.ServiceT) error
	GetTRBySrcDst(string, string) (*dm.MTraceroute, error)
	GetTRBySrcDstWithStaleness(string, string, Staleness) (*dm.MTraceroute, error)
	GetIntersectingTraceroute(string, string, Staleness) (*dm.MTraceroute, error)
}

type PingProvider interface {
	GetPingBySrcDst(string, string) (*dm.Ping, error)
	StorePing(*dm.Ping) error
}

type VantagePointProvider interface {
	SetController(string, string) error
	RemoveController(string, string) error
	UpdateVp(*dm.VantagePoint) error
	GetVpByIp(int64) (*dm.VantagePoint, error)
	GetVpByHostname(string) (*dm.VantagePoint, error)
	GetByController(string) ([]*dm.VantagePoint, error)
	GetSpoofers() ([]*dm.VantagePoint, error)
	GetTimeStamps() ([]*dm.VantagePoint, error)
	GetRecordRoute() ([]*dm.VantagePoint, error)
	UpdateCanSpoof(int64) error
	GetRecSpoof() ([]*dm.VantagePoint, error)
	GetActive() ([]*dm.VantagePoint, error)
	GetAll() ([]*dm.VantagePoint, error)
	Close() error
}
