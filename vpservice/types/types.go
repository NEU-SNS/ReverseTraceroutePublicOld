package types

import "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"

// VPProvider is the interface for a provider for vantage points
type VPProvider interface {
	GetAllVPs() ([]*pb.VantagePoint, error)
	GetRRSpoofers(target, limit uint32) ([]*pb.VantagePoint, error)
	GetTSSpoofers(target, limit uint32) ([]*pb.VantagePoint, error)
	UpdateVP(vp pb.VantagePoint) error
	GetVPsForTesting(limit int) ([]*pb.VantagePoint, error)
	UpdateActiveVPs(vps []pb.VantagePoint) error
}
