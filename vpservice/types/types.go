/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

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

// Quarantine is a vp quarantine
type Quarantine interface {
	GetVP() pb.VantagePoint
	GetReason() QuarantineReason
	GetAttempt() int
	GetAdded() time.Time
	GetLastAttempt() time.Time
	GetBackoff() time.Time
	GetInitialBackoff() time.Duration
	GetMultiplier() int
	GetMaxBackoff() time.Duration
	GetNextInitialBackoff() time.Duration
	GetExpire() time.Time
}

type defaultQuarantine struct {
	VP                 pb.VantagePoint  `json:"vp"`
	Reason             QuarantineReason `json:"reason"`
	Attempt            int              `json:"attempt"`
	Added              time.Time        `json:"added"`
	LastAttempt        time.Time        `json:"last_attempt"`
	Backoff            time.Time        `json:"backoff"`
	InitialBackoff     time.Duration    `json:"initial_backoff"`
	Multiplier         int              `json:"multiplier"`
	MaxBackoff         time.Duration    `json:"max_backoff"`
	NextInitialBackoff time.Duration    `json:"next_init_backoff"`
	Expire             time.Time        `json:"expire"`
}

func (dq *defaultQuarantine) GetVP() pb.VantagePoint {
	return dq.VP
}
func (dq *defaultQuarantine) GetReason() QuarantineReason {
	return dq.Reason
}
func (dq *defaultQuarantine) GetAttempt() int {
	return dq.Attempt
}
func (dq *defaultQuarantine) GetAdded() time.Time {
	return dq.Added
}
func (dq *defaultQuarantine) GetLastAttempt() time.Time {
	return dq.LastAttempt
}
func (dq *defaultQuarantine) GetBackoff() time.Time {
	return dq.Backoff
}
func (dq *defaultQuarantine) GetInitialBackoff() time.Duration {
	return dq.InitialBackoff
}
func (dq *defaultQuarantine) GetMultiplier() int {
	return dq.Multiplier
}
func (dq *defaultQuarantine) GetMaxBackoff() time.Duration {
	return dq.MaxBackoff
}
func (dq *defaultQuarantine) GetNextInitialBackoff() time.Duration {
	return dq.NextInitialBackoff
}
func (dq *defaultQuarantine) GetExpire() time.Time {
	return dq.Expire
}

type manualQuarantine struct {
	defaultQuarantine
}

// NewDefaultQuarantine creates a defaultQuarantine
func NewDefaultQuarantine(vp pb.VantagePoint, prevQuar bool) Quarantine {
	var q defaultQuarantine
	q.InitialBackoff = time.Hour * 24
	q.Multiplier = 2
	q.MaxBackoff = time.Hour * 24 * 7
	q.NextInitialBackoff = time.Hour * 24
	if prevQuar {
		q.InitialBackoff = q.GetNextInitialBackoff()
	}
	return &q
}

// NewManualQuarantine creates a manualQuarantine with expire exp
func NewManualQuarantine(vp pb.VantagePoint, exp time.Time) Quarantine {
	q := manualQuarantine{}
	q.Expire = exp
	return &q
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
