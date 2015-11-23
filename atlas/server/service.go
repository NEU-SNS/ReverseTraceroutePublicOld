package server

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
)

// AtlasService is the interface for the atlas
type AtlasService interface {
	GetIntersectingPath(context.Context, <-chan *dm.IntersectionRequest) (<-chan *dm.IntersectionResponse, error)
	GetPathsWithToken(context.Context, <-chan *dm.TokenRequest) (<-chan *dm.TokenResponse, error)
}
