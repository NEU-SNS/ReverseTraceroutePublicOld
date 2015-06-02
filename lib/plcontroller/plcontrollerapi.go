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
package plcontroller

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
)

const ID = "ID"

func (c *plControllerT) Ping(ctx con.Context, arg *dm.PingArg) (pr *dm.Ping, err error) {
	val := ctx.Value(ID)
	glog.Info("Ping Called for req: %s", val)
	pr = new(dm.Ping)
	*pr, err = plController.runPing(*arg)
	glog.Info("Ping done for req: %s, got: %v", val, pr)
	return
}

func (c *plControllerT) Traceroute(ctx con.Context, arg *dm.TracerouteArg) (tr *dm.Traceroute, err error) {
	val := ctx.Value(ID)
	glog.Info("Traceroute Called for req: %s", val)
	tr = new(dm.Traceroute)
	*tr, err = plController.runTraceroute(*arg)
	glog.Info("Traceroute done for req: %s, got: %v", val, tr)
	return
}

func (c *plControllerT) Stats(ctx con.Context, arg *dm.StatsArg) (sr *dm.Stats, err error) {
	val := ctx.Value(ID)
	glog.Infof("GetStats Called for req: %s", val)
	sr = new(dm.Stats)
	*sr = plController.getStats()
	return
}

func (c *plControllerT) NotifyRecSpoof(ctx con.Context, arg *dm.NotifyRecSpoof) (nr *dm.NotifyRecSpoofResponse, err error) {
	glog.Infof("Recieving notification for a recieved spoof")
	nr = new(dm.NotifyRecSpoofResponse)
	err = c.updateCanSpoof(arg.Ip)
	return
}

func (c *plControllerT) Register(ctx con.Context, arg *dm.VantagePoint) (rr *dm.RegisterResponse, err error) {
	glog.Infof("VP Registering: %v", arg)
	rr = new(dm.RegisterResponse)
	err = c.register(arg)
	return
}

func (c *plControllerT) UpdateVp(ctx con.Context, arg *dm.VantagePoint) (ur *dm.UpdateResponse, err error) {
	glog.Infof("Updateing VP: %v", arg)
	ur = new(dm.UpdateResponse)
	err = c.updateVp(arg)
	return
}

func (c *plControllerT) GetActiveVPs(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Info("Getting active VPs")
	ret = new(dm.VPReturn)
	vps, err := c.getActiveVPs()
	ret.Vps = vps
	return
}

func (c *plControllerT) GetAllVPs(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Info("Getting All VPs")
	ret = new(dm.VPReturn)
	vps, err := c.getAllVPs()
	ret.Vps = vps
	return
}

func (c *plControllerT) GetRecordRouteVPs(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Info("Getting RecordRoute VPs")
	ret = new(dm.VPReturn)
	vps, err := c.getRecordRouteVPs()
	ret.Vps = vps
	return
}

func (c *plControllerT) GetSpoofingVPs(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Info("Getting Spoofing VPs")
	ret = new(dm.VPReturn)
	vps, err := c.getSpoofingVPs()
	ret.Vps = vps
	return
}

func (c *plControllerT) GetTimeStampVPs(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Info("Getting Timestamp VPs")
	ret = new(dm.VPReturn)
	vps, err := c.getTimeStampVPs()
	ret.Vps = vps
	return
}

func (c *plControllerT) GetVP(ctx con.Context, arg *dm.VPRequest) (ret *dm.VPReturn, err error) {
	glog.Infof("Getting VP: %v", arg)
	ret = new(dm.VPReturn)
	vp, err := c.getVP(arg)
	ret.Vps = vp
	return
}
