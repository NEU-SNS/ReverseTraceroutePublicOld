package vpservice

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"golang.org/x/net/context"
)

// GRPCServ is a grpc service that satisfies the server interface
type GRPCServ struct {
	server.VPService
}

// GetVPs satisfies the VPService interface
func (gs GRPCServ) GetVPs(ctx context.Context, req *dm.VPRequest) (*dm.VPReturn, error) {
	return gs.VPService.GetVPs(ctx, req)
}
