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

// Package testdataaccess a test client that satisfies the dataaccess interfaces
package testdataaccess

import (
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

type testAccess struct{}

// NewVP creates a test data access
func NewVP() (da.VantagePointProvider, error) {
	return &testAccess{}, nil
}

func (t *testAccess) GetPingBySrcDst(string, string) (*dm.Ping, error) {
	return nil, nil
}
func (t *testAccess) StorePing(*dm.Ping) error {
	return nil
}

func (t *testAccess) StoreTraceroute(*dm.Traceroute, dm.ServiceT) error {
	return nil
}
func (t *testAccess) GetTRBySrcDst(string, string) (*dm.MTraceroute, error) {
	return nil, nil
}
func (t *testAccess) GetTRBySrcDstWithStaleness(string, string, da.Staleness) (*dm.MTraceroute, error) {
	return nil, nil
}
func (t *testAccess) GetIntersectingTraceroute(string, string, da.Staleness) (*dm.MTraceroute, error) {
	return nil, nil
}

func (t *testAccess) SetController(string, string) error {
	return nil
}
func (t *testAccess) RemoveController(string, string) error {
	return nil
}
func (t *testAccess) UpdateVp(*dm.VantagePoint) error {
	return nil
}
func (t *testAccess) GetVpByIP(int64) (*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetVpByHostname(string) (*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetByController(string) ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetSpoofers() ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetTimeStamps() ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetRecordRoute() ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) UpdateCanSpoof(int64) error {
	return nil
}
func (t *testAccess) GetRecSpoof() ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetActive() ([]*dm.VantagePoint, error) {
	return nil, nil
}
func (t *testAccess) GetAll() ([]*dm.VantagePoint, error) {
	return nil, nil
}

func (t *testAccess) Close() error {
	return nil
}
