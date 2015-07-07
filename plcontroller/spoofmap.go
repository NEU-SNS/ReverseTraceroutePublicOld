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

//Package plcontroller is the library for creating a planet-lab controller
package plcontroller

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type sender interface {
	Send(interface{}) error
}

type spoofMap struct {
	spoofs map[uint32]dm.Spoof
	rec    chan interface{}
	reg    chan dm.Spoof
	quit   chan interface{}
}

func newSpoofMap() *spoofMap {
	sps := make(map[uint32]dm.Spoof)
	recChan := make(chan interface{}, 20)
	regChan := make(chan dm.Spoof, 20)
	qc := make(chan interface{})
	return &spoofMap{spoofs: sps, rec: recChan, reg: regChan, quit: qc}
}

func (s *spoofMap) Register(sp dm.Spoof) {
	s.reg <- sp
}

func (s *spoofMap) Quit() {
	close(s.quit)
}

func (s *spoofMap) Receive(sp interface{}) {
	s.rec <- sp
}

func (s *spoofMap) monitor() {
	for {
		select {
		case <-s.quit:
			return
		case sp := <-s.reg:
			s.spoofs[sp.Id] = sp
		case rec := <-s.rec:
			continue
		}
	}
}
