package hdclient

import (
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/rescrv/HyperDex/bindings/go/client"
	"sync"
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
	return &hdClient{c: c, config: con, eChan: echan}, nil
}

func (c *hdClient) StoreTraceroute(t *dm.Traceroute) error {

}
