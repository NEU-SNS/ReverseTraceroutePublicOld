package dataaccess

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"time"
)

type DataProvider interface {
	ServiceProvider
	TracerouteProvider
	PingProvider
	Close() error
}

type ServiceProvider interface {
	GetServices() ([]*dm.Service, error)
}

type Staleness time.Duration

type TracerouteProvider interface {
	StoreTraceroute(*dm.Traceroute) error
	GetTRBySrcDst(string, string) (*dm.Traceroute, error)
	GetTRBySrcDstWithStaleness(string, string, Staleness) (*dm.Traceroute, error)
}

type PingProvider interface {
	GetPingBySrcDst(string, string) (*dm.Ping, error)
	StorePing(*dm.Ping) error
}
