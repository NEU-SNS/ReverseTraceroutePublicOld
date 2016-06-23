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

// QuarantineReason is a reason to quarantine a vp
type QuarantineReason int

const (
	// CantPerformMeasurement is the QuarantineReason when a vp can't perform
	// any measurements
	CantPerformMeasurement QuarantineReason = iota
	// Manual is when some quarantines a VP manually for a set amount of time
	Manual
)

var (
	reasonToDesc = map[QuarantineReason]string{
		CantPerformMeasurement: "VP can not perform any measurements",
		Manual:                 "VP was manually quarantined",
	}
)

// Quarantine represents the quarntining of a vantange point
type Quarantine struct {
	vp          pb.VantagePoint
	Site        string
	Hostname    string
	IP          uint32
	Reason      QuarantineReason
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
func NewQuarantineFromVP(vp pb.VantagePoint, reason QuarantineReason) Quarantine {

	return Quarantine{

		Site:     vp.Site,
		Hostname: vp.Hostname,
		IP:       vp.Ip,
		Reason:   reason,
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
