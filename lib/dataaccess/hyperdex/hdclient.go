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
package hdclient

import (
	"fmt"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/lib/util"
	"github.com/golang/glog"
	"github.com/rescrv/HyperDex/bindings/go/client"
	"time"
)

var (
	ErrorWrongType = fmt.Errorf("The data stored is of the wrong type")
	ErrorTooOld    = fmt.Errorf("The stored object is too old")
)

type hdClient struct {
	c      *client.Client
	config dm.DbConfig
	eChan  chan client.Error
}

func New(con dm.DbConfig) (*hdClient, error) {
	c, e, echan := client.NewClient(con.Host, con.Port)
	if e != nil {
		return nil, e
	}
	go func() {
		for {
			select {
			case e := <-echan:
				glog.Exitf("Hyperdex Client error: %v", e)
			}
		}
	}()
	return &hdClient{c: c, config: con, eChan: echan}, nil
}

var ErrorInvalidIP = fmt.Errorf("Invalid IP in traceroute")

func (c *hdClient) StoreTraceroute(tr *dm.Traceroute, s dm.ServiceT) error {
	glog.Infof("Storing traceroute: %v", *tr)
	t := tr.GetStart()
	var st time.Time
	if t == nil {
		st = time.Now()
	} else {
		st = time.Unix(t.Sec, util.MicroToNanoSec(t.Usec))
	}
	hops := tr.GetHops()
	if hops == nil || len(hops) == 0 {
		return nil
	}
	src, err := util.IpStringToInt32(tr.Src)
	if err != nil {
		return ErrorInvalidIP
	}
	dst, err := util.IpStringToInt32(tr.Dst)
	if err != nil {
		return ErrorInvalidIP
	}
	hlist := make([]int64, len(hops))
	for i, hop := range hops {
		ip, err := util.IpStringToInt32(hop.Addr)
		if err != nil {
			return fmt.Errorf("Invalid IP address in hop: %v", hop)
		}
		hlist[i] = int64(ip)
	}
	key := fmt.Sprintf("%d:%d", src, dst)

	e := c.c.Put(c.config.TracerouteSpace,
		key,
		client.Attributes{"src": src,
			"dst":     dst,
			"service": s,
			"date":    st.Unix(),
			"route":   client.ListInt(hlist)})

	if e != nil {
		return e
	}
	glog.Infof("Traceroute stored")
	for i, hop := range hlist {
		key := fmt.Sprintf("%d:%d", hop, dst)
		e = c.c.Put(c.config.TraceHopSpace,
			key,
			client.Attributes{"src": src,
				"hop":     hop,
				"dst":     dst,
				"service": s,
				"date":    st.Unix(),
				"route":   client.ListInt(hlist[i:])})

		if e != nil {
			return e
		}
	}
	glog.Infof("Traceroute hop stored")
	return nil
}

func makeMTraceroute(a *client.Attributes) (*dm.MTraceroute, error) {
	glog.Infof("Making traceroute from: %v", a)
	tr := new(dm.MTraceroute)
	hl := (*a)["route"]
	if d, ok := (*a)["date"].(int64); ok {
		tr.Date = d
	}

	src := (*a)["src"]
	dst := (*a)["dst"]

	if ssrc, ok := src.(int64); ok {
		s, err := util.Int32ToIpString(uint32(ssrc))
		if err != nil {
			glog.Errorf("Failed to parse src string from: %d", ssrc)
			return nil, ErrorWrongType
		}
		tr.Src = s
	}

	if sdst, ok := dst.(int64); ok {
		s, err := util.Int32ToIpString(uint32(sdst))
		if err != nil {
			glog.Errorf("Failed to parse dst string from: %d", dst)
			return nil, ErrorWrongType
		}
		tr.Dst = s
	}

	if hop, ok := (*a)["hop"]; ok {
		if h, ok := hop.(int64); ok {
			hi, err := util.Int32ToIpString(uint32(h))
			if err != nil {
				glog.Errorf("Failed to parse hop string from: %d", h)
				return nil, ErrorWrongType
			}
			tr.Hop = hi
		}
	}

	if serv, ok := (*a)["service"].(int64); ok {
		if serv < 0 {
			glog.Errorf("Failed to parse service: %d", serv)
			return nil, ErrorWrongType
		}
		tr.Service = dm.ServiceT(serv)
	} else {
		glog.Errorf("Failed to parse service: %d", serv)
		return nil, ErrorWrongType
	}
	if h, ok := hl.(client.ListInt); ok {
		tr.Hops = make([]uint32, 0)
		for val := range []int64(h) {
			tr.Hops = append(tr.Hops, uint32(val))
		}
		return tr, nil
	}
	glog.Errorf("Failed to parse hop list")
	return nil, ErrorWrongType
}

func (c *hdClient) GetTRBySrcDst(src, dst string) (*dm.MTraceroute, error) {
	return c.GetTRBySrcDstWithStaleness(src, dst, -1)
}

func (c *hdClient) GetTRBySrcDstWithStaleness(src, dst string, s da.Staleness) (*dm.MTraceroute, error) {
	nsrc, err := util.IpStringToInt32(src)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	ndst, err := util.IpStringToInt32(dst)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	key := fmt.Sprintf("%d:%d", nsrc, ndst)
	attrs, e := c.c.Get(c.config.TracerouteSpace, key)
	if e != nil {
		return nil, e
	}
	tr, err := makeMTraceroute(&attrs)
	if err != nil {
		return nil, err
	}
	sasi := int64(s)
	if sasi > 0 && time.Now().Unix() > tr.Date+sasi {
		return nil, ErrorTooOld
	}
	glog.Infof("Got traceroute from db: %v", tr)
	return tr, nil
}

func (c *hdClient) GetIntersectingTraceroute(hop, dst string, s da.Staleness) (*dm.MTraceroute, error) {
	nhop, err := util.IpStringToInt32(hop)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	ndst, err := util.IpStringToInt32(dst)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	key := fmt.Sprintf("%d:%d", nhop, ndst)
	attrs, e := c.c.Get(c.config.TraceHopSpace, key)
	if e != nil {
		return nil, e
	}
	tr, err := makeMTraceroute(&attrs)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (c *hdClient) GetPingBySrcDst(src, dst string) (*dm.Ping, error) {
	return nil, nil
}

func (c *hdClient) StorePing(p *dm.Ping) error {
	return nil
}

func (c *hdClient) Close() error {
	c.c.Destroy()
	return nil
}

func (c *hdClient) SetController(ip, myip string) error {
	nip, err := util.IpStringToInt32(ip)
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	e := c.c.Put(c.config.VantagePointSpace, myip,
		client.Attributes{"controller": nip, "last_updated": ts})
	return e
}

func (c *hdClient) RemoveController(ip, myip string) error {
	nip, err := util.IpStringToInt32(ip)
	if err != nil {
		return err
	}
	nmyip, err := util.IpStringToInt32(myip)
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	e := c.c.CondPut(c.config.VantagePointSpace, ip,
		[]client.Predicate{
			client.Predicate{"ip", nip, client.EQUALS},
			client.Predicate{"controller", nmyip, client.EQUALS},
		},
		client.Attributes{"controller": int64(-1),
			"last_updated": ts})
	return e
}

func makeVpAttributes(vp *dm.VantagePoint) (client.Attributes, error) {
	var sshable, sudoProblem,
		recordRoute,
		lastUpdate, spoof,
		ts, active, rec_spoof int64

	if vp.Sshable {
		sshable = 1
	}
	if vp.SudoProblem {
		sudoProblem = 1
	}
	if vp.RecordRoute {
		recordRoute = 1
	}
	if vp.Spoof {
		spoof = 1
	}
	if vp.Ts {
		ts = 1
	}
	if vp.Active {
		active = 1
	}
	if vp.RecSpoof {
		rec_spoof = 1
	}
	ip, err := util.IpStringToInt32(vp.Ip)
	if err != nil {
		return client.Attributes{}, err
	}
	lastUpdate = time.Now().Unix()
	attr := client.Attributes{
		"sshable":      sshable,
		"ip":           int64(ip),
		"sudo_problem": sudoProblem,
		"record_route": recordRoute,
		"spoof":        spoof,
		"ts":           ts,
		"active":       active,
		"last_update":  lastUpdate,
		"rec_spoof":    rec_spoof,
	}
	return attr, nil
}

var keys = []string{"hostname",
	"ip",
	"sshable",
	"sudo_problem",
	"record_route",
	"ts",
	"active",
	"controller",
	"spoof",
	"last_updated",
	"hostname",
	"rec_spoof"}

func attrToVp(attr client.Attributes) (*dm.VantagePoint, error) {
	vp := new(dm.VantagePoint)
	for _, key := range keys {
		if val, ok := attr[key]; ok {
			switch key {
			case "ip":
				if ip, ok := val.(int64); ok {
					sip, err := util.Int32ToIpString(uint32(ip))
					if err != nil {
						return nil, err
					}
					vp.Ip = sip
				}
			case "sshable":
				if sshable, ok := val.(int64); ok {
					if sshable > 0 {
						vp.Sshable = true
					}
				}
			case "sudo_problem":
				if sp, ok := val.(int64); ok {
					if sp > 0 {
						vp.SudoProblem = true
					}
				}
			case "record_route":
				if rr, ok := val.(int64); ok {
					if rr > 0 {
						vp.RecordRoute = true
					}
				}
			case "ts":
				if ts, ok := val.(int64); ok {
					if ts > 0 {
						vp.Ts = true
					}
				}
			case "active":
				if active, ok := val.(int64); ok {
					if active > 0 {
						vp.Active = true
					}
				}
			case "controller":
				if cont, ok := val.(int64); ok {
					c, err := util.Int32ToIpString(uint32(cont))
					if err != nil {
						return nil, err
					}
					vp.Controller = c
				}
			case "spoof":
				if spoof, ok := val.(int64); ok {
					if spoof > 0 {
						vp.Spoof = true
					}
				}
			case "last_updated":
				if lu, ok := val.(int64); ok {
					vp.LastUpdated = lu
				}
			case "hostname":
				if hn, ok := val.(string); ok {
					vp.Hostname = hn
				}
			case "rec_spoof":
				if rs, ok := val.(int64); ok {
					if rs > 0 {
						vp.RecSpoof = true
					}
				}
			}
		}
	}
	return vp, nil
}

func (c *hdClient) UpdateVp(vp *dm.VantagePoint) error {
	attr, err := makeVpAttributes(vp)
	if err != nil {
		return err
	}
	e := c.c.Put(c.config.VantagePointSpace, vp.Ip, attr)
	return e
}

func (c *hdClient) GetVpByIp(ip uint32) (*dm.VantagePoint, error) {
	ips, err := util.Int32ToIpString(ip)
	if err != nil {
		return nil, err
	}
	attr, e := c.c.Get(c.config.VantagePointSpace, ips)
	if e.Status == client.NOTFOUND {
		return nil, e
	}
	return attrToVp(attr)
}

func (c *hdClient) GetVpByHostname(hn string) (*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"hostname", hn, client.EQUALS}})
	vps, err := vpsFromChannels(attrs, errs)
	if err != nil {
		return nil, err
	}
	if len(vps) != 1 {
		return nil, fmt.Errorf("Non-unique hostname")
	}
	return vps[0], nil
}

func vpsFromChannels(ac chan client.Attributes, ec chan client.Error) ([]*dm.VantagePoint, error) {
	vps := make([]*dm.VantagePoint, 0)
	for attr := range ac {
		a, err := attrToVp(attr)
		if err != nil {
			return nil, err
		}
		vps = append(vps, a)
	}

	for e := range ec {
		return nil, e
	}
	return vps, nil
}

func (c *hdClient) GetByController(cont string) ([]*dm.VantagePoint, error) {
	nc, err := util.IpStringToInt32(cont)
	if err != nil {
		return nil, err
	}
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"controller", nc, client.EQUALS}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) GetSpoofers() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"spoof", 0, client.GREATER_THAN}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) GetTimeStamps() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"ts", 0, client.GREATER_THAN}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) GetRecordRoute() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"record_route", 0, client.GREATER_THAN}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) GetActive() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"active", 0, client.GREATER_THAN}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) GetRecSpoof() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{client.Predicate{"rec_spoof", 0, client.GREATER_THAN}})

	return vpsFromChannels(attrs, errs)
}

func (c *hdClient) UpdateCanSpoof(ip uint32) error {
	ips, err := util.Int32ToIpString(ip)
	if err != nil {
		return err
	}
	e := c.c.Put(c.config.VantagePointSpace, ips,
		client.Attributes{"spoof": 1})
	if e != nil {
		return e
	}
	return nil
}

func (c *hdClient) GetAll() ([]*dm.VantagePoint, error) {
	attrs, errs := c.c.Search(c.config.VantagePointSpace,
		[]client.Predicate{})
	return vpsFromChannels(attrs, errs)
}
