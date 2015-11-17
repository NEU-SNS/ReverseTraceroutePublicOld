package server

import (
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"golang.org/x/net/context"
)

// VPService is the service that handles VPs
type VPService interface {
	GetVPs(context.Context, *dm.VPRequest) (*dm.VPReturn, error)
}
