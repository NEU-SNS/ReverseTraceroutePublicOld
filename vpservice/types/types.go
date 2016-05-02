package types

import "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"

// VPProvider is the interface for a provider for vantage points
type VPProvider interface {
	GetVPs() ([]*pb.VantagePoint, error)
	GetRRSpoofers(target, limit uint32) ([]RRVantagePoint, error)
	GetTSSpoofers(target, limit uint32) ([]TSVantagePoint, error)
	UpdateVP(vp pb.VantagePoint) error
	GetVPsForTesting(limit int) ([]*pb.VantagePoint, error)
	UpdateActiveVPs(vps []*pb.VantagePoint) error
}

// RRVantagePoint represents a vantage point
// used for spoofed RR probes
type RRVantagePoint struct {
	pb.VantagePoint
	Dist   uint32
	Target uint32
}

// TSVantagePoint represents a vantage point
// used for spoofed TS probes
type TSVantagePoint struct {
	pb.VantagePoint
	Target uint32
}

// Config represents the config options for the revtr service.
type Config struct {
	RootCA   *string `flag:"root-ca"`
	CertFile *string `flag:"cert-file"`
	KeyFile  *string `flag:"key-file"`
}

// NewConfig creats a Config
func NewConfig() Config {
	return Config{
		RootCA:   new(string),
		CertFile: new(string),
		KeyFile:  new(string),
	}
}
