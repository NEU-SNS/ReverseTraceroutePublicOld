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
	"fmt"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

var (
	// ErrorIDInUse is returned when the id of a spoofed request is already in use.
	ErrorIDInUse = fmt.Errorf("The is received is already in use.")
	// ErrorSpoofNotFound is returned when a spoof is received that doesn't have
	// have a matching id
	ErrorSpoofNotFound = fmt.Errorf("Received a spoof with no matching Id")
)

// Sender is the interface for something that can sent a slice of SpoofedProbes
// to an address
type Sender interface {
	Send([]dm.Probe, string) error
}

type spoof struct {
	S    dm.Spoof
	Time time.Time
}

type spoofMap struct {
	spoofs    map[uint32]spoof
	rec       chan dm.Probe
	reg       chan dm.Spoof
	regErr    chan error
	recErr    chan error
	send      chan dm.Probe
	quit      chan interface{}
	transport Sender
}

func newSpoofMap(s Sender) (sm *spoofMap) {
	sps := make(map[uint32]spoof)
	regChan := make(chan dm.Spoof, 20)
	recChan := make(chan dm.Probe, 20)
	sendChan := make(chan dm.Probe, 100)
	errChan := make(chan error)
	regeChan := make(chan error)
	qc := make(chan interface{})

	sm = &spoofMap{
		spoofs:    sps,
		rec:       recChan,
		reg:       regChan,
		quit:      qc,
		regErr:    regeChan,
		recErr:    errChan,
		send:      sendChan,
		transport: s,
	}
	go sm.monitor()
	go sm.sendSpoofs()
	return
}

func (s *spoofMap) Register(sp dm.Spoof) error {
	s.reg <- sp
	return <-s.regErr
}

func (s *spoofMap) Quit() {
	close(s.quit)
}

func (s *spoofMap) register(sp dm.Spoof) error {
	if curr, ok := s.spoofs[sp.Id]; ok {
		if time.Since(curr.Time) < time.Second*60 {
			return ErrorIDInUse
		}
	}
	s.spoofs[sp.Id] = spoof{S: sp, Time: time.Now()}
	return nil
}

func (s *spoofMap) receive(sp dm.Probe) error {
	if spoof, ok := s.spoofs[sp.Id]; ok {
		delete(s.spoofs, sp.Id)
		sp.SenderIp = spoof.S.Ip
		s.send <- sp
		return nil
	}
	return ErrorSpoofNotFound
}

func (s *spoofMap) Receive(sp dm.Probe) error {
	s.rec <- sp
	return <-s.recErr
}

func (s *spoofMap) sendSpoofs() {
	probes := make(map[string][]dm.Probe)
	for {
		select {
		case <-s.quit:
			return
		case <-time.After(time.Second):
			for ip := range probes {
				go func(ps []dm.Probe, addr string) {
					s.transport.Send(ps, addr)
				}(probes[ip], ip)
			}
			probes = make(map[string][]dm.Probe)
		case sp := <-s.rec:
			probes[sp.SenderIp] = append(probes[sp.SenderIp], sp)
		}
	}
}

func (s *spoofMap) monitor() {
	for {
		select {
		case <-s.quit:
			return
		case sp := <-s.reg:
			s.regErr <- s.register(sp)
		case rec := <-s.rec:
			s.recErr <- s.receive(rec)
		}
	}
}
