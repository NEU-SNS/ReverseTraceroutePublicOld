package client

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	pb.VPServiceClient
}

// VPSource is the inteface to something that gives vps
type VPSource interface {
	GetVPs() (*dm.VPReturn, error)
}

// New returns a VPSource
func New(ctx context.Context, cc *grpc.ClientConn) VPSource {
	return client{Context: ctx, VPServiceClient: pb.NewVPServiceClient(cc)}
}

func (c client) GetVPs() (*dm.VPReturn, error) {
	vpr, err := c.VPServiceClient.GetVPs(c.Context, &dm.VPRequest{})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return vpr, nil
}
