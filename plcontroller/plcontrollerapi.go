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
	// ErrorTimeout is returned when a measurement times out
	ErrorTimeout = fmt.Errorf("Measurement timed out")
)

const id = "ID"

func (c *plControllerT) Ping(pa *dm.PingArg, stream plc.PLController_PingServer) error {
	pings := pa.GetPings()
	if len(pings) == 0 {
		return ErrorEmptyArgList
	}
	doneChan := make(chan struct{})
	errChan := make(chan error, len(pings))
	quitChan := make(chan struct{})
	var wg sync.WaitGroup
	for _, ping := range pings {
		wg.Add(1)
		go func(st plc.PLController_PingServer, w *sync.WaitGroup, p *dm.PingMeasurement) {
			glog.Infof("Rinning ping: %v", p)
			defer wg.Done()
			sendChan := make(chan struct{})
			var pp *dm.Ping
			for {
				select {
				case <-quitChan:
					return
				case <-sendChan:
					glog.Infof("Sending ping: %v", pp)
					if e := st.Send(pp); e != nil {
						errChan <- e
						close(quitChan)
					}
					return
				default:
					pr, err := c.runPing(p)
					glog.Infof("Got ping result: %v, with error %v", pr, err)
					if err != nil {
						pr.Error = err.Error()
					}
					pp = &pr
					close(sendChan)
				}
			}
		}(stream, &wg, ping)
	}

	go func() {
		wg.Wait()
		close(doneChan)
	}()
	select {
	case <-doneChan:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *plControllerT) Traceroute(ta *dm.TracerouteArg, stream plc.PLController_TracerouteServer) error {
	traces := ta.GetTraceroutes()
	if len(traces) == 0 {
		return ErrorEmptyArgList
	}
	doneChan := make(chan struct{})
	errChan := make(chan error, len(traces))
	quitChan := make(chan struct{})
	var wg sync.WaitGroup
	for _, trace := range traces {
		wg.Add(1)
		go func(st plc.PLController_TracerouteServer, w *sync.WaitGroup, t *dm.TracerouteMeasurement) {
			defer wg.Done()
			sendChan := make(chan struct{})
			var ttr *dm.Traceroute
			for {
				select {
				case <-quitChan:
					return
				case <-sendChan:
					if e := st.Send(ttr); e != nil {
						errChan <- e
						close(quitChan)
					}
					return
				default:
					tr, err := c.runTraceroute(t)
					if err != nil {
						tr.Error = err.Error()
					}
					ttr = &tr
					close(sendChan)
				}
			}
		}(stream, &wg, trace)
	}

	go func() {
		wg.Wait()
		close(doneChan)
	}()
	select {
	case <-doneChan:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *plControllerT) ReceiveSpoof(ctx con.Context, arg *dm.RecSpoof) (ret *dm.NotifyRecSpoofResponse, err error) {
	ret = nil
	err = nil
	return
}

func (c *plControllerT) NotifyRecSpoof(ctx con.Context, arg *dm.RecSpoof) (nr *dm.NotifyRecSpoofResponse, err error) {
	glog.Infof("Recieving notification for a recieved spoof")
	nr = new(dm.NotifyRecSpoofResponse)
	return
}

func (c *plControllerT) GetVPs(vpr *dm.VPRequest, stream plc.PLController_GetVPsServer) error {
	glog.Info("Getting All VPs")

	return nil
}
