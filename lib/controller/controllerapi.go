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
)

func (c ControllerApi) Register(arg int, reply *int) error {
	*reply = 5
	return nil
}

func (c ControllerApi) Ping(arg dm.PingArg, ret *dm.PingReturn) error {
	glog.Info("Handling Ping Request")
	marg := dm.MArg{Service: arg.Service, SArg: arg}
	mr, err := controller.handleMeasurement(&marg, dm.PING)
	ret.Status = mr.Status
	ret.Dur = mr.Dur
	if ping, ok := mr.SRet.(*dm.Ping); ok {
		ret.Ping = *ping
	}
	return err
}

func (c ControllerApi) Traceroute(arg dm.MArg, ret *dm.MReturn) error {
	mr, err := controller.handleMeasurement(&arg, dm.TRACEROUTE)
	*ret = *mr
	return err
}

func (c ControllerApi) GetStats(arg dm.StatsArg, ret *dm.StatsReturn) error {
	glog.Info("Handling Stats Request")
	marg := dm.MArg{Service: arg.Service, SArg: arg}
	mr, err := controller.handleMeasurement(&marg, dm.STATS)
	ret.Status = mr.Status
	ret.Dur = mr.Dur
	if stats, ok := mr.SRet.(*dm.Stats); ok {
		ret.Stats = *stats
	}
	return err
}
