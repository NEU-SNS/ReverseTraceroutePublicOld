package server

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
)

// AtlasService is the interface for the atlas
type AtlasService interface {
	GetIntersectingPath(context.Context, *dm.IntersectionRequest) ([]*dm.IntersectionResponse, error)
	GetPathsWithToken(context.Context, *dm.TokenRequest) ([]*dm.TokenResponse, error)
}
