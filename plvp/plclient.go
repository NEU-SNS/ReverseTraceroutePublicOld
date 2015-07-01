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
package plvp

import (
	"fmt"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type plClient struct {
	cc     *grpc.ClientConn
	client plc.PLControllerClient
}

func (c *plClient) disconnect() error {
	if c.cc != nil {
		cl := c.cc
		c.cc = nil
		return cl.Close()
	}
	return fmt.Errorf("Called disconnect on unconnected plClient")
}

func (c *plClient) connect(addr string, timeout time.Duration) error {
	glog.Infof("Trying to connect to: %s", addr)
	cc, err := grpc.Dial(addr, grpc.WithTimeout(timeout))
	if err != nil {
		glog.Errorf("PlClient Failed to connect: %v", err)
		return err
	}
	c.cc = cc
	c.client = plc.NewPLControllerClient(cc)
	return nil
}

func (c *plClient) Register(addr string,
	to time.Duration,
	vp *dm.VantagePoint) error {

	glog.Infof("Registering VP")
	err := c.connect(addr, to)
	if err != nil {
		return err
	}
	defer c.disconnect()
	_, err = c.client.Register(con.Background(), vp)
	return err
}

func (c *plClient) NotifyRecSpoof(addr string,
	to time.Duration,
	rs *dm.NotifyRecSpoof) error {

	glog.Infof("Notifying rec of spoofed ping from: %v", rs)
	err := c.connect(addr, to)
	if err != nil {
		return err
	}
	defer c.disconnect()
	_, err = c.client.NotifyRecSpoof(con.Background(), rs)
	return err
}

func (c *plClient) UpdateVp(addr string,
	to time.Duration,
	vp *dm.VantagePoint) error {

	glog.Infof("Updating VP with: %v", vp)
	err := c.connect(addr, to)
	if err != nil {
		return err
	}
	defer c.disconnect()
	_, err = c.client.UpdateVp(con.Background(), vp)
	return err
}
