package controller

import (
	"errors"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/nu7hatch/gouuid"
	"time"
)

const (
	IP                            = 0
	PORT                          = 1
	GenRequest     MRequestState  = "generating request"
	RequestRoute   MRequestState  = "routing request"
	ExecuteRequest MRequestState  = "executing request"
	SUCCESS        MRequestStatus = "SUCCESS"
	ERROR          MRequestStatus = "ERROR"
	PING           dm.MType       = "PING"
	TRACEROUTE     dm.MType       = "TRACEROUTE"
)

var (
	ErrorInvalidIP       = errors.New("invalid IP address passed to Start")
	ErrorServiceNotFound = errors.New("service not found")
)

type MRequestStatus string
type MRequestState string
type ControllerApi struct{}
type RoutedRequest func() (*MReturn, error)

type MArg struct {
	Service string
	SArg    interface{}
	Src     string
	Dst     string
}

type Request struct {
	Id    *uuid.UUID
	Stime time.Time
	Dur   time.Duration
	Args  interface{}
	Key   string
	Type  dm.MType
}

type PingArg struct {
}

type MReturn struct {
	Status MRequestStatus
	SRet   interface{}
}

type PingReturn struct {
}

type MRequestError struct {
	cause    MRequestState
	causeErr error
}
