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

func (c *hdClient) GetServices() ([]*dm.Service, error) {
	return []*dm.Service{&dm.Service{
		IPAddr: []string{"129.10.113.205:45000"},
		Key:    dm.ServiceT_PLANET_LAB,
	}}, nil
}

func New(con dm.DbConfig) (da.DataProvider, error) {
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
	src, err := util.IpStringToInt64(tr.Src)
	if err != nil {
		return ErrorInvalidIP
	}
	dst, err := util.IpStringToInt64(tr.Dst)
	if err != nil {
		return ErrorInvalidIP
	}
	hlist := make([]int64, len(hops))
	for i, hop := range hops {
		ip, err := util.IpStringToInt64(hop.Addr)
		if err != nil {
			return fmt.Errorf("Invalid IP address in hop: %v", hop)
		}
		hlist[i] = ip
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
		s, err := util.Int64ToIpString(ssrc)
		if err != nil {
			glog.Errorf("Failed to parse src string from: %d", ssrc)
			return nil, ErrorWrongType
		}
		tr.Src = s
	}

	if sdst, ok := dst.(int64); ok {
		s, err := util.Int64ToIpString(sdst)
		if err != nil {
			glog.Errorf("Failed to parse dst string from: %d", dst)
			return nil, ErrorWrongType
		}
		tr.Dst = s
	}

	if hop, ok := (*a)["hop"]; ok {
		if h, ok := hop.(int64); ok {
			hi, err := util.Int64ToIpString(h)
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
		tr.Hops = []int64(h)
		return tr, nil
	}
	glog.Errorf("Failed to parse hop list")
	return nil, ErrorWrongType
}

func (c *hdClient) GetTRBySrcDst(src, dst string) (*dm.MTraceroute, error) {
	return c.GetTRBySrcDstWithStaleness(src, dst, -1)
}

func (c *hdClient) GetTRBySrcDstWithStaleness(src, dst string, s da.Staleness) (*dm.MTraceroute, error) {
	nsrc, err := util.IpStringToInt64(src)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	ndst, err := util.IpStringToInt64(dst)
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
	nhop, err := util.IpStringToInt64(hop)
	if err != nil {
		return nil, ErrorInvalidIP
	}
	ndst, err := util.IpStringToInt64(dst)
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
	nip, err := util.IpStringToInt64(ip)
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	e := c.c.Put(c.config.VantagePointSpace, myip,
		client.Attributes{"controller": nip, "last_updated": ts})
	return e
}

func (c *hdClient) RemoveController(ip, myip string) error {
	nip, err := util.IpStringToInt64(ip)
	if err != nil {
		return err
	}
	nmyip, err := util.IpStringToInt64(myip)
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
		ip, recordRoute,
		lastUpdate, spoof,
		ts, active int64

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
	ip, err := util.IpStringToInt64(vp.Ip)
	if err != nil {
		return client.Attributes{}, err
	}
	lastUpdate = time.Now().Unix()
	attr := client.Attributes{
		"sshable":      sshable,
		"ip":           ip,
		"sudo_problem": sudoProblem,
		"record_route": recordRoute,
		"spoof":        spoof,
		"ts":           ts,
		"active":       active,
		"last_update":  lastUpdate,
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
	"last_updated"}

func attrToVp(attr client.Attributes) (*dm.VantagePoint, error) {
	vp := new(dm.VantagePoint)
	for _, key := range keys {
		if val, ok := attr[key]; ok {
			switch key {
			case "ip":
				if ip, ok := val.(int64); ok {
					sip, err := util.Int64ToIpString(ip)
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
					c, err := util.Int64ToIpString(cont)
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

func (c *hdClient) GetVpByIp(ip string) (*dm.VantagePoint, error) {
	attr, e := c.c.Get(c.config.VantagePointSpace, ip)
	if e.Status == client.NOTFOUND {
		return nil, e
	}
	return attrToVp(attr)
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
	nc, err := util.IpStringToInt64(cont)
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
