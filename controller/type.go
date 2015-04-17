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
 DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package controller

import (
	"errors"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/nu7hatch/gouuid"
	"time"
)

const (
	IP                            = 0
	PORT                          = 1
	GenRequest     MRequestState  = "generating request"
	RequestRoute   MRequestState  = "routing request"
	ExecuteRequest MRequestState  = "executing request"
	SUCCESS        MRequestStatus = "SUCCESS"
	ERROR          MRequestStatus = "ERROR"
	PING           dm.MType       = "PING"
	TRACEROUTE     dm.MType       = "TRACEROUTE"
)

var (
	ErrorInvalidIP       = errors.New("invalid IP address passed to Start")
	ErrorServiceNotFound = errors.New("service not found")
)

type MRequestStatus string
type MRequestState string
type ControllerApi struct{}
type RoutedRequest func() (*MReturn, error)

type MArg struct {
	Service string
	SArg    interface{}
	Src     string
	Dst     string
}

type Request struct {
	Id    *uuid.UUID
	Stime time.Time
	Dur   time.Duration
	Args  interface{}
	Key   string
	Type  dm.MType
}

type PingArg struct {
}

type MReturn struct {
	Status MRequestStatus
	SRet   interface{}
}

type PingReturn struct {
}

type MRequestError struct {
	cause    MRequestState
	causeErr error
}
