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
	"errors"
	"time"

	"code.google.com/p/go-uuid/uuid"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	con "golang.org/x/net/context"
)

var (
	ErrorServiceNotFound         = errors.New("service not found")
	ErrorMeasurementToolNotFound = errors.New("measurement tool not found")
)

type MeasurementTool interface {
	Ping(con.Context, *dm.PingArg) (*dm.Ping, error)
	Traceroute(con.Context, *dm.TracerouteArg) (*dm.Traceroute, error)
	Stats(con.Context, *dm.StatsArg) (*dm.Stats, error)
	GetVP(con.Context, *dm.VPRequest) (*dm.VPReturn, error)
	Connect(string, time.Duration) error
}

type RoutedRequest func() (*dm.MReturn, Request, error)

type Request struct {
	Id    uuid.UUID
	Stime time.Time
	Dur   time.Duration
	Args  interface{}
	Key   dm.ServiceT
	Type  dm.MType
}

type Flags struct {
	Local      LocalConfig
	ConfigPath string
	Db         dm.DbConfig
}

type Config struct {
	Local LocalConfig
	Db    dm.DbConfig
}

type LocalConfig struct {
	Addr         string
	CloseStdDesc bool
	PProfAddr    string
	Proto        string
	AutoConnect  bool
	SecureConn   bool
	CertFile     string
	KeyFile      string
	ConnTimeout  int64
	Services     []*dm.Service
}
