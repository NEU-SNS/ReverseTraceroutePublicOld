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
package datamodel

import (
	"fmt"
	"time"
)

//http://stackoverflow.com/questions/10210188/instance-new-type-golang
type Creator func() interface{}

var TypeMap map[string]Creator

func init() {
	TypeMap = make(map[string]Creator, 10)
	TypeMap["Stats"] = createStats
	TypeMap["Ping"] = createPing
	TypeMap["Traceroute"] = createTraceroute
}

const (
	GenRequest     MRequestState = "generating request"
	RequestRoute   MRequestState = "routing request"
	ExecuteRequest MRequestState = "executing request"
)

type MRequestState string

type MArg struct {
	Service ServiceT
	SArg    interface{}
}

type ServiceArg struct {
	Service ServiceT
}

type MReturn struct {
	Status MRequestStatus
	Dur    time.Duration
	SRet   interface{}
}

func (m MRequestError) Error() string {
	return fmt.Sprintf("Error occured while %s caused by: %v", m.Cause, m.CauseErr)
}

type MRequestError struct {
	Cause    MRequestState
	CauseErr error
}
