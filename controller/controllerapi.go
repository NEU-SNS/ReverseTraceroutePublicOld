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
	"io"
	"time"

	cont "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	con "golang.org/x/net/context"
)

func (c *controllerT) Ping(pa *dm.PingArg, stream cont.Controller_PingServer) error {
	pms := pa.GetPings()
	if pms == nil {
		return nil
	}
	start := time.Now()
	ctx, cancel := con.WithCancel(stream.Context())
	defer cancel()
	res := c.doPing(ctx, pms)
	for {
		select {
		case p, ok := <-res:
			if !ok {
				end := time.Since(start).Seconds()
				pingResponseTimes.Observe(end)
				return nil
			}
			if err := stream.Send(p); err != nil {
				log.Error(err)
				end := time.Since(start).Seconds()
				pingResponseTimes.Observe(end)
				return err
			}
		case <-ctx.Done():
			end := time.Since(start).Seconds()
			pingResponseTimes.Observe(end)
			return ctx.Err()
		}
	}
}

func (c *controllerT) Traceroute(ta *dm.TracerouteArg, stream cont.Controller_TracerouteServer) error {
	tms := ta.GetTraceroutes()
	if tms == nil {
		return nil
	}
	start := time.Now()
	ctx, cancel := con.WithCancel(stream.Context())
	defer cancel()
	res := c.doTraceroute(ctx, tms)
	for {
		select {
		case t, ok := <-res:
			if !ok {
				end := time.Since(start).Seconds()
				tracerouteResponseTimes.Observe(end)
				return nil
			}
			if err := stream.Send(t); err != nil {
				log.Error(err)
				end := time.Since(start).Seconds()
				tracerouteResponseTimes.Observe(end)
				return err
			}
		case <-ctx.Done():
			end := time.Since(start).Seconds()
			tracerouteResponseTimes.Observe(end)
			return ctx.Err()
		}
	}
}

func (c *controllerT) GetVPs(ctx con.Context, gvp *dm.VPRequest) (vpr *dm.VPReturn, err error) {
	vpr, err = c.doGetVPs(ctx, gvp)
	return
}

func (c *controllerT) ReceiveSpoofedProbes(probes cont.Controller_ReceiveSpoofedProbesServer) error {
	log.Debug("ReceiveSpoofedProbes")
	for {
		pr, err := probes.Recv()
		if err == io.EOF {
			return probes.SendAndClose(&dm.ReceiveSpoofedProbesResponse{})
		}
		if err != nil {
			log.Error(err)
			return err
		}
		c.doRecSpoof(probes.Context(), pr)
	}
}
