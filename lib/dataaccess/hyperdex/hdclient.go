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
	return nil
}
