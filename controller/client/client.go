package client

import (
	"github.com/NEU-SNS/ReverseTraceroute/controllerapi"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type client struct {
	context.Context
	controllerapi.ControllerClient
}

// Client is a client for the controller
type Client interface {
	Ping(*datamodel.PingArg) (controllerapi.Controller_PingClient, error)
	Traceroute(*datamodel.TracerouteArg) (controllerapi.Controller_TracerouteClient, error)
	GetVps(*datamodel.VPRequest) (*datamodel.VPReturn, error)
	ReceiveSpoofedProbes() (controllerapi.Controller_ReceiveSpoofedProbesClient, error)
}

// New creates a new controller client
func New(ctx context.Context, cc *grpc.ClientConn) Client {
	return client{Context: ctx, ControllerClient: controllerapi.NewControllerClient(cc)}
}

func (c client) Ping(pa *datamodel.PingArg) (controllerapi.Controller_PingClient, error) {
	return c.ControllerClient.Ping(c.Context, pa)
}

func (c client) Traceroute(ta *datamodel.TracerouteArg) (controllerapi.Controller_TracerouteClient, error) {
	return c.ControllerClient.Traceroute(c.Context, ta)
}

func (c client) GetVps(vpr *datamodel.VPRequest) (*datamodel.VPReturn, error) {
	return c.ControllerClient.GetVPs(c.Context, vpr)
}

func (c client) ReceiveSpoofedProbes() (controllerapi.Controller_ReceiveSpoofedProbesClient, error) {
	return c.ControllerClient.ReceiveSpoofedProbes(c.Context)
}
