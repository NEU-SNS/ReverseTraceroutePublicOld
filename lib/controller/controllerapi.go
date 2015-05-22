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
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
	con "golang.org/x/net/context"
)

func (c *controllerT) Ping(ctx con.Context, pa *dm.PingArg) (pr *dm.PingReturn, err error) {
	glog.Info("Handling Ping Request")
	pr = new(dm.PingReturn)
	pr, err = c.doPing(ctx, pa)
	return
}

func (c *controllerT) Stats(ctx con.Context, sa *dm.StatsArg) (sr *dm.StatsReturn, err error) {
	glog.Info("Handling Stats Request")
	sr = new(dm.StatsReturn)
	sr, err = c.doStats(ctx, sa)
	return
}

func (c *controllerT) Traceroute(ctx con.Context, ta *dm.TracerouteArg) (tr *dm.TracerouteReturn, err error) {
	glog.Info("Handling Traceroute Request")
	tr = new(dm.TracerouteReturn)
	tr, err = c.doTraceroute(ctx, ta)
	return
}
