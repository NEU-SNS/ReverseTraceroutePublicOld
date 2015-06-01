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
	StoreTraceroute(*dm.Traceroute, dm.ServiceT) error
	GetTRBySrcDst(string, string) (*dm.MTraceroute, error)
	GetTRBySrcDstWithStaleness(string, string, Staleness) (*dm.MTraceroute, error)
	GetIntersectingTraceroute(string, string, Staleness) (*dm.MTraceroute, error)
}

type PingProvider interface {
	GetPingBySrcDst(string, string) (*dm.Ping, error)
	StorePing(*dm.Ping) error
}

type VantagePointProvider interface {
	SetController(string, string) error
	RemoveController(string, string) error
	UpdateVp(*dm.VantagePoint) error
	GetVpByIp(string) (*dm.VantagePoint, error)
	GetByController(string) ([]*dm.VantagePoint, error)
	GetSpoofers() ([]*dm.VantagePoint, error)
	GetTimeStamps() ([]*dm.VantagePoint, error)
	GetRecordRoute() ([]*dm.VantagePoint, error)
	GetRecSpoof() ([]*dm.VantagePoint, error)
	GetActive() ([]*dm.VantagePoint, error)
	Close() error
}
