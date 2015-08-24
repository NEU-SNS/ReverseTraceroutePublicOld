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
	"sync"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
)

var (
	// ErrorEmptyArgList is returned when a measurement request comes in with an
	// empty list of args
	ErrorEmptyArgList = fmt.Errorf("Empty argument list.")
	// ErrorNilArgList is returned when a measurement request comes in with a nil
	// argument list
	ErrorNilArgList = fmt.Errorf("Nil argument list.")
	// ErrorTimeout is returned when a measurement times out
	ErrorTimeout = fmt.Errorf("Measurement timed out.")
)

func (c *plControllerT) Ping(pa *dm.PingArg, stream plc.PLController_PingServer) error {
	pings := pa.GetPings()
	if pings == nil {
		return ErrorNilArgList
	}
	if len(pings) == 0 {
		return ErrorEmptyArgList
	}
	doneChan := make(chan struct{})
	sendChan := make(chan *dm.Ping, len(pings))
	var wg sync.WaitGroup
	for _, ping := range pings {
		wg.Add(1)
		go func(p *dm.PingMeasurement) {
			defer wg.Done()
			pr, err := c.runPing(p)
			if err != nil {
				glog.Infof("Got ping result: %v, with error %v", pr, err)
				pr.Error = err.Error()
			}
			sendChan <- &pr
		}(ping)
	}

	go func() {
		wg.Wait()
		close(doneChan)
	}()
	for {
		select {
		case <-doneChan:
			return nil
		case p := <-sendChan:
			if err := stream.Send(p); err != nil {
				return err
			}
		}
	}

}

func (c *plControllerT) Traceroute(ta *dm.TracerouteArg, stream plc.PLController_TracerouteServer) error {
	traces := ta.GetTraceroutes()
	if traces == nil {
		return ErrorNilArgList
	}
	if len(traces) == 0 {
		return ErrorEmptyArgList
	}
	doneChan := make(chan struct{})
	sendChan := make(chan *dm.Traceroute, len(traces))
	var wg sync.WaitGroup
	for _, trace := range traces {
		wg.Add(1)
		go func(t *dm.TracerouteMeasurement) {
			defer wg.Done()
			tr, err := c.runTraceroute(t)
			if err != nil {
				glog.Infof("Got tracerotue result: %v, with error %v", tr, err)
				tr.Error = err.Error()
			}
			sendChan <- &tr
		}(trace)
	}

	go func() {
		wg.Wait()
		close(doneChan)
	}()
	for {
		select {
		case <-doneChan:
			return nil
		case t := <-sendChan:
			if err := stream.Send(t); err != nil {
				return err
			}
		}
	}
}

func (c *plControllerT) ReceiveSpoof(rs *dm.RecSpoof, stream plc.PLController_ReceiveSpoofServer) error {
	spoofs := rs.GetSpoofs()
	if spoofs == nil {
		return ErrorNilArgList
	}
	if len(spoofs) == 0 {
		return ErrorEmptyArgList
	}
	for _, spoof := range spoofs {
		ret, err := c.recSpoof(spoof)
		if err != nil {
			ret.Error = err.Error()
		}
		if e := stream.Send(ret); e != nil {
			return e
		}
	}
	return nil
}

func (c *plControllerT) AcceptProbes(ctx con.Context, probes *dm.SpoofedProbes) (*dm.SpoofedProbesResponse, error) {
	ps := probes.GetProbes()
	if ps == nil {
		return nil, ErrorNilArgList
	}
	if len(ps) == 0 {
		return nil, ErrorEmptyArgList
	}
	for _, p := range ps {
		c.acceptProbe(p)
	}
	return &dm.SpoofedProbesResponse{}, nil
}

func (c *plControllerT) GetVPs(vpr *dm.VPRequest, stream plc.PLController_GetVPsServer) error {
	glog.Info("Getting All VPs")
	vps, err := c.db.GetVPs()
	if err != nil {
		return nil
	}
	if err := stream.Send(&dm.VPReturn{Vps: vps}); err != nil {
		return err
	}
	return nil
}
