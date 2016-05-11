package types

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

// IntersectionQuery represents a request to the TRStore for an intersecting traceroute
type IntersectionQuery struct {
	Addr, Dst, Src uint32
	Alias          bool
	Stale          time.Duration
	IgnoreSource   bool
}

// TRStore is the interface required by the
type TRStore interface {
	FindIntersectingTraceroute(IntersectionQuery) (*pb.Path, error)
	StoreAtlasTraceroute(*datamodel.Traceroute) error
	GetAtlasSources(uint32, time.Duration) ([]uint32, error)
}
