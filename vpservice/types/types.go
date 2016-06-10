package types

import (
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
)

// VPProvider is the interface for a provider for vantage points
type VPProvider interface {
	GetVPs() ([]*pb.VantagePoint, error)
	GetRRSpoofers(target uint32) ([]RRVantagePoint, error)
	GetTSSpoofers(target uint32) ([]TSVantagePoint, error)
	UpdateVP(vp pb.VantagePoint) error
	GetVPsForTesting(limit int) ([]*pb.VantagePoint, error)
	UpdateActiveVPs(vps []*pb.VantagePoint) ([]*pb.VantagePoint, []*pb.VantagePoint, error)
	UnquarantineVPs(vps []Quarantine) error
	QuarantineVPs(vps []Quarantine) error
	GetQuarantined() ([]Quarantine, error)
}

// Quarantine represents the quarntining of a vantange point
type Quarantine struct {
	Site        string
	Hostname    string
	IP          uint32
	Reason      string
	Attempt     int
	Added       time.Time
	LastAttempt time.Time
	Retry       int64
	tried       bool
}

const (
	minQuarantine = time.Hour * 24 * 2
	maxQuarantine = time.Hour * 24 * 7
)

// NewQuarantineFromVP creates a quarantine from the given vp
// with default settings with the given reason
func NewQuarantineFromVP(vp pb.VantagePoint, reason string) Quarantine {

	return Quarantine{
		Site:     vp.Site,
		Hostname: vp.Hostname,
		IP:       vp.Ip,
		Reason:   reason,
		Attempts: 0,
	}
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
