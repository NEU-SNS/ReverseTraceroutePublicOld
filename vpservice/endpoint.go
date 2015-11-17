package vpservice

import (
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/server"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
)

// MakeGetVPsEndpoint makes an endpoint
func MakeGetVPsEndpoint(svc server.VPService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*datamodel.VPRequest)
		ret, err := svc.GetVPs(ctx, req)
		if err != nil {
			return nil, err
		}
		return ret, nil
	}
}
