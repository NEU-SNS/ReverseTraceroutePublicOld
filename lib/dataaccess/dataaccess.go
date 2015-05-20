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
	"encoding/json"
	"fmt"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
	"github.com/rescrv/HyperDex/bindings/go/client"
	"time"
)

var (
	ErrorCacheValueNotFound = fmt.Errorf("Cached value not found")
)

type DataAccess interface {
	GetServices() ([]*dm.Service, error)
	StoreTraceroute(t *dm.Traceroute) error
	GetTraceroute(src, dst string, s time.Duration) (*dm.Traceroute, error)
	GetPing(src, dst string, s time.Duration) (*dm.Ping, error)
	StorePing(*dm.Ping) error
	Destroy()
}

type dataAccess struct {
	c    *client.Client
	conf dm.DbConfig
	ec   chan client.Error
}

func (d *dataAccess) GetPing(src, dst string, s time.Duration) (*dm.Ping, error) {
	key := fmt.Sprintf("%s:%s", src, dst)
	glog.Infof("Trying to get ping for: %s", key)
	res, err := d.c.Get(d.conf.PingSpace, key)
	if err != nil {
		return nil, convertErr(err)
	}
	ping := new(dm.Ping)
	obj := res[d.conf.PingAttr]
	if doc, ok := obj.(client.Document); ok {
		err := json.Unmarshal([]byte(doc.Doc)[:len(doc.Doc)-1], ping)
		if err != nil {
			glog.Errorf("GetPing failed to unmarshal json: %s with err: %v",
				doc.Doc, err)
			return nil, err
		}
		start := time.Unix(ping.Start.Sec, ping.Start.USec)
		d := time.Since(start)

		if d > s {
			ping = nil
		}
		return ping, nil
	}
	return nil, nil
}

func (d *dataAccess) StorePing(p *dm.Ping) error {
	p.Prestore()
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%s", p.Src, p.Dst)
	glog.Infof("Storing ping: %s %s", key, b)
	err = d.c.Put(d.conf.PingSpace, key,
		client.Attributes{d.conf.PingAttr: client.Document{string(b)}})
	return err
}

func (d *dataAccess) GetServices(ip string) []*dm.Service {
	return []*dm.Service{&dm.Service{
		IPAddr: []string{"127.0.0.1:45000"},
		Key:    dm.ServiceT_PLANET_LAB,
	}}
}

func (d *dataAccess) StoreTraceroute(t *dm.Traceroute) error {
	t.Prestore()
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%s", t.Src, t.Dst)
	glog.Infof("Storing traceroute: %s %s", key, b)
	err = d.c.Put(d.conf.TracerouteSpace, key,
		client.Attributes{d.conf.TracerouteAttr: client.Document{string(b)}})

	return err
}

func convertErr(err *client.Error) (e error) {
	switch err.Status {
	case client.NOTFOUND:
		e = ErrorCacheValueNotFound
	default:
		e = err
	}
	return
}

func (d *dataAccess) GetTraceroute(src, dst string, s time.Duration) (*dm.Traceroute, error) {
	key := fmt.Sprintf("%s:%s", src, dst)
	glog.Infof("Trying to get ping for: %s", key)
	res, err := d.c.Get(d.conf.TracerouteSpace, key)
	if err != nil {
		return nil, convertErr(err)
	}
	tr := new(dm.Traceroute)
	obj := res[d.conf.TracerouteAttr]
	if doc, ok := obj.(client.Document); ok {
		err := json.Unmarshal([]byte(doc.Doc)[:len(doc.Doc)-1], tr)
		if err != nil {
			glog.Errorf("GetTraceroute failed to unmarshal json: %s with err: %v",
				doc.Doc, err)
			return nil, err
		}
		start := time.Unix(tr.Start.Sec, tr.Start.USec)
		d := time.Since(start)

		if d > s {
			tr = nil
		}
		return tr, nil
	}
	return nil, nil
}

func (d *dataAccess) GetServices() ([]*dm.Service, error) {
	rchan, errChan := d.c.Search(d.conf.ServiceSpace, nil)
	servs := make([]*dm.Service, 0)
	for res := range rchan {
		var serv dm.Service
		obj := res[d.conf.ServiceAttr]
		if doc, ok := obj.(client.Document); ok {
			err := json.Unmarshal([]byte(doc.Doc)[:len(doc.Doc)-1], &serv)
			if err != nil {
				glog.Errorf("GetServices failed to unmarshal json: %s with err: %v",
					doc.Doc, err)
				return nil, err
			}
			servs = append(servs, &serv)
		}
		return servs, nil
	}

	for e := range errChan {
		return nil, e
	}
	panic("should never get here GetServices")
	return nil, nil
}

func (d *dataAccess) Destroy() {
	d.c.Destroy()
}

func (d *dataAccess) handleErrors() {
	go func() {
		for e := range d.ec {
			glog.Errorf("DB Client issue: %v", e)
		}
	}()
}

func New(conf dm.DbConfig) (DataAccess, error) {
	glog.Infof("Connecting to database: %v", conf)
	c, e, err := client.NewClient(conf.Host, conf.Port)
	if e != nil {
		return nil, e
	}
	da := &dataAccess{conf: conf, c: c, ec: err}
	da.handleErrors()
	return da, nil
}
