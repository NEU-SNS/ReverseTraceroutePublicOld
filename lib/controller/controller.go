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
     * Neither the name of the University of Washington nor the
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
package controller

import (
	"code.google.com/p/go-uuid/uuid"
	"errors"
	"fmt"
	da "github.com/NEU-SNS/ReverseTraceroute/lib/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
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
	router   Router
	requests uint64
	time     time.Duration
}

var controller controllerT

func parseAddrArg(addr string) (int, net.IP, error) {
	parts := strings.Split(addr, ":")
	ip := parts[IP]
	if ip == "localhost" {
		ip = "127.0.0.1"
	}
	port := parts[PORT]
	pport, err := strconv.Atoi(port)
	if err != nil {
		glog.Errorf("Failed to parse port")
		return 0, nil, err
	}
	if pport < 1 || pport > 65535 {
		glog.Errorf("Invalid port passed to Start: %d", pport)
		return 0, nil, ErrorInvalidPort
	}
	pip := net.ParseIP(ip)
	if pip == nil {
		glog.Errorf("Invalid IP passed to Start: %s", ip)
		return 0, nil, ErrorInvalidIP
	}
	return pport, pip, nil
}

func Start(n, laddr string, db da.DataAccess) chan error {
	errChan := make(chan error, 1)
	if db == nil {
		glog.Errorf("Nil db in Controller Start")
		errChan <- errors.New("Controller Start, nil DB")
		return errChan
	}
	controller.ptype = n
	controller.db = db
	controller.router = createRouter()
	port, ip, err := parseAddrArg(laddr)
	if err != nil {
		glog.Errorf("Failed to start Controller")
		errChan <- err
		return errChan
	}
	controller.ip = ip
	controller.port = port
	go startRpc(n, laddr, errChan)
	return errChan
}

func (m MRequestError) Error() string {
	return fmt.Sprintf("Error occured while %s caused by: %v", m.cause, m.causeErr)
}

func makeErrorReturn(cause MRequestState, err error) (*MReturn, error) {
	return &MReturn{Status: ERROR}, MRequestError{cause: cause, causeErr: err}
}

func (c controllerT) handleMeasurement(arg *MArg, mt dm.MType) (*MReturn, error) {
	r, err := generateRequest(arg, mt)
	if err != nil {
		glog.Errorf("Error generating request: %v", err)
		return makeErrorReturn(GenRequest, err)
	}
	rr, err := controller.routeRequest(r)
	if err != nil {
		glog.Errorf("%s: Failed to route request: %v, with error: %v", r.Id, r, err)
		return makeErrorReturn(RequestRoute, err)
	}
	result, err := rr()
	if err != nil {
		glog.Errorf("%s: Failed to execute request: %v, with error: %v", r.Id, rr, err)
		return makeErrorReturn(ExecuteRequest, err)
	}
	return result, nil
}

func (c controllerT) routeRequest(r Request) (RoutedRequest, error) {
	rr, err := c.router.RouteRequest(r)
	if err != nil {
		return nil, err
	}
	return rr, err
}

func generateRequest(marg *MArg, mt dm.MType) (Request, error) {
	id := uuid.NewRandom()
	r := Request{
		Id:   id,
		Args: marg,
		Key:  marg.Service,
		Type: mt}
	glog.Infof("%s: Generated Request: %v", id, r)
	return r, nil
}

func startRpc(n, laddr string, eChan chan error) error {
	api := new(ControllerApi)
	server := rpc.NewServer()
	server.Register(api)
	l, e := net.Listen(n, laddr)
	if e != nil {
		glog.Errorf("Failed to listen: %v", e)
		eChan <- e
	}
	glog.Infof("Controller started, listening on: %s", laddr)
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorf("Accept failed: %v", err)
			eChan <- err
			continue
		}
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
