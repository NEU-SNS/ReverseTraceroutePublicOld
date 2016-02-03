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
	GetIntersectingPath(context.Context) (pb.Atlas_GetIntersectingPathClient, error)
	GetPathsWithToken(context.Context) (pb.Atlas_GetPathsWithTokenClient, error)
}

// New returns a new atlas
func New(ctx context.Context, cc *grpc.ClientConn) Atlas {
	return client{Context: ctx, AtlasClient: pb.NewAtlasClient(cc)}
}

// GetIntersectingPath gets an intersecting path
func (c client) GetIntersectingPath(ctx context.Context) (pb.Atlas_GetIntersectingPathClient, error) {
	return c.AtlasClient.GetIntersectingPath(ctx)
}

// GetPathsWithToken gets a path from a token
func (c client) GetPathsWithToken(ctx context.Context) (pb.Atlas_GetPathsWithTokenClient, error) {
	return c.AtlasClient.GetPathsWithToken(ctx)
}
