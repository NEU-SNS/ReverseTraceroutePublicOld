package controller

import (
	"errors"
	"fmt"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/golang/glog"
	"github.com/nu7hatch/gouuid"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
	"time"
)

type controllerT struct {
	port     int
	ip       net.IP
	ptype    string
	db       da.DataAccess
	router   Router
	requests uint64
	time     time.Duration
}

var (
	controller     controllerT
	ErrorInvalidIP = errors.New("Invalid IP address passed to Start")
)

const (
	IP   = 0
	PORT = 1
)

type MRequestStatus string
type MRequestState string

const (
	GenRequest     MRequestState = "generating request"
	RequestRoute   MRequestState = "routing request"
	ExecuteRequest MRequestState = "executing request"
)

const (
	SUCCESS MRequestStatus = "SUCCESS"
	ERROR   MRequestStatus = "ERROR"
)

func parseAddrArg(addr string) (int, net.IP, error) {
	parts := strings.Split(addr, ":")
	ip := parts[IP]
	port := parts[PORT]
	pport, err := strconv.Atoi(port)
	if err != nil {
		glog.Errorf("Failed to parse port")
		return 0, nil, err
	}
	pip := net.ParseIP(ip)
	if pip == nil {
		glog.Errorf("Invalid IP passed to Start")
		return 0, nil, ErrorInvalidIP
	}
	return pport, pip, nil
}

func Start(n, laddr string, db da.DataAccess) chan error {
	errChan := make(chan error, 1)
	if db == nil {
		glog.Fatalf("Nil db in Controller Start")
	}
	controller.ptype = n
	controller.db = db
	controller.router = createRouter()
	port, ip, err := parseAddrArg(laddr)
	if err != nil {
		glog.Errorf("Failed to start Controller")
		errChan <- err
	}
	controller.ip = ip
	controller.port = port
	go startRpc(n, laddr, errChan)
	return errChan
}

type ControllerApi int

type MArg struct {
	Service string
	SArg    interface{}
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

func (m MRequestError) Error() string {
	return fmt.Sprintf("Error occured while %s caused by: %v", m.cause, m.causeErr)
}

func (c ControllerApi) Register(arg int, reply *int) error {
	glog.Info("Register Called")
	*reply = 5
	return nil
}

func (c ControllerApi) Ping(arg MArg, ret *MReturn) error {
	mr, err := controller.handleMeasurement(&arg)
	ret = mr
	return err
}

func (c ControllerApi) Traceroute(arg MArg, ret *MReturn) error {
	mr, err := controller.handleMeasurement(&arg)
	ret = mr
	return err
}

func (c controllerT) handleMeasurement(arg *MArg) (*MReturn, error) {
	r, err := generateRequest(arg)
	if err != nil {
		return &MReturn{Status: ERROR}, MRequestError{cause: GenRequest, causeErr: err}
	}
	rr, err := controller.routeRequest(r)
	if err != nil {
		return &MReturn{Status: ERROR}, MRequestError{cause: RequestRoute, causeErr: err}
	}
	rChan, err := rr()
	if err != nil {
		return &MReturn{Status: ERROR}, MRequestError{cause: ExecuteRequest, causeErr: err}
	}
	return <-rChan, nil
}

type Request struct {
	Id    *uuid.UUID
	Stime time.Time
	Dur   time.Duration
	Args  interface{}
	Key   string
}

type RoutedRequest func() (chan *MReturn, error)

func (c controllerT) routeRequest(r Request) (RoutedRequest, error) {
	rr, err := c.router.RouteRequest(r)
	if err != nil {
		return nil, err
	}
	return rr, err
}

func generateRequest(marg *MArg) (Request, error) {
	id, err := uuid.NewV4()
	if err != nil {
		glog.Errorf("Failed to generate UUID: %v", err)
		return Request{}, err
	}
	return Request{
		Id:   id,
		Args: marg,
		Key:  marg.Service}, nil
}

func startRpc(n, laddr string, eChan chan error) {
	api := new(ControllerApi)
	server := rpc.NewServer()
	server.Register(api)
	l, e := net.Listen(n, laddr)
	if e != nil {
		glog.Fatalln("Failed to listen: %v", e)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Fatalf("Accept failed: %v", err)
		}
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
