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
package dataaccess

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type dataAccess struct {
	conf dm.DbConfig
}

func uToNSec(u int64) int64 {
	//1000 nsec to a usec
	return u * 1000
}

func (d *dataAccess) StoreTraceroute(t *dm.Traceroute, s dm.ServiceT) error {
	return nil
}

func (d *dataAccess) GetTRBySrcDst(src, dst string) (*dm.MTraceroute, error) {
	return nil, nil
}

func (d *dataAccess) GetTRBySrcDstWithStaleness(src, dst string, s Staleness) (*dm.MTraceroute, error) {
	return nil, nil
}

func (d *dataAccess) GetPingBySrcDst(src, dst string) (*dm.Ping, error) {
	return nil, nil
}

func (d *dataAccess) StorePing(p *dm.Ping) error {
	return nil
}

func (d *dataAccess) Close() error {
	return nil
}

/*
func New(c dm.DbConfig) (DataProvider, error) {
	return &dataAccess{conf: c}, nil
}
*/