package atlas

import (
	"github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
)

// Atlas is the atlas
type Atlas struct {
	da dataaccess.DataAccess
}

// GetIntersectingPath satisfies the server interface
func (a *Atlas) GetIntersectingPath(ctx context.Context, in <-chan *dm.IntersectionRequest) (<-chan *dm.IntersectionResponse, error) {
	return nil, nil
}

// GetPathsWithToken satisfies the server interface
func (a *Atlas) GetPathsWithToken(ctx context.Context, in <-chan *dm.TokenRequest) (<-chan *dm.TokenResponse, error) {
	return nil, nil
}
