package client

import (
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	pb.AtlasClient
}

// Atlas is the atlas
type Atlas interface {
	GetIntersectingPath() (pb.Atlas_GetIntersectingPathClient, error)
}

// New returns a new atlas
func New(ctx context.Context, cc *grpc.ClientConn) Atlas {
	return client{Context: ctx, AtlasClient: pb.NewAtlasClient(cc)}
}

// GetIntersectingPath sets an intersecting path
func (c client) GetIntersectingPath() (pb.Atlas_GetIntersectingPathClient, error) {
	return c.AtlasClient.GetIntersectingPath(c.Context)
}
