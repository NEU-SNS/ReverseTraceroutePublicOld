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
	"time"
)

func TestStart(t *testing.T) {
	eChan := Start("tcp", "localhost:45000", da.New())

	select {
	case e := <-eChan:
		t.Errorf("TestStart failed %v", e)
	case <-time.After(time.Second * 2):

	}

}

func TestStartNoDB(t *testing.T) {
	eChan := Start("tcp", "localhost:45000", nil)

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Error("Controller started with nil DB")
	}

}

func TestStartInvalidIP(t *testing.T) {
	eChan := Start("tcp", "-1:45000", da.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Errorf("TestStartInvalidIP no error thrown with invalid ip")
	}

}

func TestStartInvalidPort(t *testing.T) {
	eChan := Start("tcp", "localhost:PORT", da.New())

	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Errorf("TestStartInvalidPort no error thrown with invalid port")
	}

}

func TestGenerateRequest(t *testing.T) {
	_, err := generateRequest(&MArg{Service: "TEST"}, PING)
	if err != nil {
		t.Errorf("TestGenerateRequest failed error: %v", err)
	}
}
