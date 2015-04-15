package controller

import (
	"errors"
	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	"github.com/NEU-SNS/ReverseTraceroute/router"
	"github.com/golang/glog"
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
	router   router.Router
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

func Start(n, laddr string, db da.DataAccess, r router.Router) chan error {
	errChan := make(chan error, 1)
	if db == nil || r == nil {
		glog.Fatalf("Nil paramter in Controller Start")
	}
	controller.ptype = n
	controller.db = db
	controller.router = r
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
	SArg interface{}
}

type PingArg struct {
}

type MReturn struct {
	SRet interface{}
}

type PingReturn struct {
}

func (c ControllerApi) Register(arg int, reply *int) error {
	glog.Info("Register Called")
	*reply = 5
	return nil
}

func (c ControllerApi) Ping(arg MArg, ret *MReturn) error {

	return nil
}

func (c ControllerApi) Traceroute(arg MArg, ret *MReturn) error {
	return nil
}

type Request struct {
	Stime time.Time
	Etime time.Time
	Oargs *Marg
	Key   string
}

func (c controllerT) routeRequest(r Request) {

}

func generateRequest(marg *MArg) Request {

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
