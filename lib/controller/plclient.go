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
package controller

import (
	"fmt"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/lib/plcontrollerapi"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

type plClient struct {
	cc       *grpc.ClientConn
	connOpen bool
	client   plc.PLControllerClient
	addr     string
}

func (c *plClient) disconnect() error {
	if c.cc != nil {
		return c.cc.Close()
	}
	return fmt.Errorf("Called disconnect on unconnected plClient")
}

func (c *plClient) Connect(addr string, timeout time.Duration) error {
	glog.Infof("Trying to connect to: %s", addr)
	if c.connOpen && addr == c.addr {
		return nil
	}
	if c.connOpen && addr != c.addr {
		err := c.cc.Close()
		if err != nil {
			return err
		}
	}
	cc, err := grpc.Dial(addr, grpc.WithTimeout(timeout))
	if err != nil {
		glog.Errorf("PlClient Failed to connect: %v", err)
		return err
	}
	c.cc = cc
	c.client = plc.NewPLControllerClient(cc)
	c.connOpen = true
	return nil
}

func (c *plClient) Ping(ctx con.Context, pa *dm.PingArg) (*dm.Ping, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.Ping(ctx, pa)
}

func (c *plClient) Traceroute(ctx con.Context, ta *dm.TracerouteArg) (*dm.Traceroute, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.Traceroute(ctx, ta)
}

func (c *plClient) Stats(ctx con.Context, s *dm.StatsArg) (*dm.Stats, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.Stats(ctx, s)
}

func (c *plClient) GetVP(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetVP(ctx, arg)
}

func (c *plClient) GetAllVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetAllVPs(ctx, arg)
}

func (c *plClient) GetActiveVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetActiveVPs(ctx, arg)
}

func (c *plClient) GetRecordRouteVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetRecordRouteVPs(ctx, arg)
}

func (c *plClient) GetTimeStampVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetTimeStampVPs(ctx, arg)
}

func (c *plClient) GetSpoofingVPs(ctx con.Context, arg *dm.VPRequest) (*dm.VPReturn, error) {
	if !c.connOpen {
		return nil, fmt.Errorf("PLClient not connected")
	}
	return c.client.GetSpoofingVPs(ctx, arg)
}
