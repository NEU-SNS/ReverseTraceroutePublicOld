package datamodel

import (
	"fmt"
	"time"
)

const (
	GenRequest     MRequestState  = "generating request"
	RequestRoute   MRequestState  = "routing request"
	ExecuteRequest MRequestState  = "executing request"
	SUCCESS        MRequestStatus = "SUCCESS"
	ERROR          MRequestStatus = "ERROR"
	PING           MType          = "PING"
	TRACEROUTE     MType          = "TRACEROUTE"
)

type MRequestStatus string
type MRequestState string

type Stats struct {
	StartTime  time.Time
	UpTime     time.Duration
	Requests   int64
	AvgReqTime time.Duration
	TotReqTime time.Duration
}

type MArg struct {
	Service string
	SArg    interface{}
	Src     string
	Dst     string
}

type PingArg struct {
}

type MReturn struct {
	Status MRequestStatus
	SRet   interface{}
	Dur    time.Duration
}

type PingReturn struct {
}

func (m MRequestError) Error() string {
	return fmt.Sprintf("Error occured while %s caused by: %v", m.Cause, m.CauseErr)
}

type MRequestError struct {
	Cause    MRequestState
	CauseErr error
}
