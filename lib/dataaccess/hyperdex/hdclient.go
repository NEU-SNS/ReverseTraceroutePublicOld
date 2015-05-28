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

type hdClient struct {
	c      *client.Client
	config dm.DbConfig
	eChan  chan client.Error
}

func (c *hdClient) GetServices() ([]*dm.Service, error) {
	return []*dm.Service{&dm.Service{
		IPAddr: []string{"127.0.0.1:45000"},
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

func microToNanoSec(usec int64) int64 {
	return usec * 1000
}

var ErrorInvalidIP = fmt.Errorf("Invalid IP in traceroute")

func (c *hdClient) StoreTraceroute(tr *dm.Traceroute) error {
	t := tr.GetStart()
	var st time.Time
	if t == nil {
		st = time.Now()
	} else {
		st = time.Unix(t.Sec, microToNanoSec(t.Usec))
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

	e := c.c.Put(c.config.TracerouteSpace, key, client.Attributes{"src": src, "dst": dst, "date": st.Unix(), "hops": hlist})
	if e != nil {
		return e
	}
	for i, hop := range hlist {
		key := fmt.Sprintf("%d:%d", hop, dst)
		e = c.c.Put(c.config.TraceHopSpace, key, client.Attributes{"hop": hop, "dst": dst, "date": st.Unix(), "hops": hlist[i:]})
		if e != nil {
			return e
		}
	}
	return nil
}
