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

//Package plcontroller is the library for creating a planet-lab controller
package plcontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	da "github.com/NEU-SNS/ReverseTraceroute/dataaccess"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/mproc"
	"github.com/NEU-SNS/ReverseTraceroute/mproc/proc"
	plc "github.com/NEU-SNS/ReverseTraceroute/plcontrollerapi"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"gopkg.in/fsnotify.v1"
)

type plControllerT struct {
	startTime time.Time
	spid      int
	server    *grpc.Server
	config    Config
	sc        scamper.Config
	mp        mproc.MProc
	db        da.VantagePointProvider
	w         *fsnotify.Watcher
	conf      Config
	client    Client
	spoofs    *spoofMap
}

// Client is the measurment client interface
// TODO: Remove interface dependency on scamper
type Client interface {
	AddSocket(*scamper.Socket)
	RemoveSocket(string)
	GetSocket(string) (*scamper.Socket, error)
	DoMeasurement(string, interface{}) (<-chan scamper.Response, error)
}

func handleScamperStop(err error, ps *os.ProcessState, p *proc.Process) bool {
	switch err.(type) {
	default:
		return false
	case *os.PathError:
		return true
	}

}

var plController plControllerT

func (c *plControllerT) recSpoof(rs *dm.RecSpoof) (*dm.NotifyRecSpoofResponse, error) {

	return nil, nil
}

func (c *plControllerT) runPing(pa *dm.PingMeasurement) (dm.Ping, error) {
	glog.Infof("Running ping for: %v", pa)
	timeout := pa.Timeout
	if timeout == 0 {
		timeout = c.conf.Local.Timeout
	}
	ret := dm.Ping{}

	resp, err := c.client.DoMeasurement(pa.Src, pa)
	if err != nil {
		return ret, err
	}

	select {
	case r := <-resp:
		err := decodeResponse(r.Bytes(), &ret)
		if err != nil {
			return ret, fmt.Errorf("Could not decode ping response: %v", err)
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		return ret, fmt.Errorf("Ping timed out")
	}
	glog.Infof("Ping done: %v", ret)
	return ret, nil
}

func (c *plControllerT) runTraceroute(ta *dm.TracerouteMeasurement) (dm.Traceroute, error) {
	glog.Infof("Running traceroute for: %v", ta)
	timeout := ta.Timeout
	if timeout == 0 {
		timeout = c.conf.Local.Timeout
	}
	ret := dm.Traceroute{}

	resp, err := c.client.DoMeasurement(ta.Src, ta)
	if err != nil {
		return ret, err
	}

	select {
	case r := <-resp:
		err := decodeResponse(r.Bytes(), &ret)
		if err != nil {
			return ret, fmt.Errorf("Could not decode traceroute response: %v", err)
		}
	case <-time.After(time.Second * time.Duration(timeout)):
		return ret, fmt.Errorf("Traceroute timed out")
	}

	glog.Infof("Traceroute done: %v", ret)
	return ret, nil
}

func decodeResponse(res []byte, ret interface{}) error {
	return json.NewDecoder(bytes.NewReader(res)).Decode(ret)
}

func convertWarts(path string, b []byte) ([]byte, error) {
	glog.Info("Converting Warts")
	res, err := util.ConvertBytes(path, b)
	if err != nil {
		glog.Errorf("Failed to converte bytes: %v", err)
		return []byte{}, err
	}
	glog.Infof("Results of converting: %s", res)
	return res, err
}

func (c *plControllerT) addSocket(sock *scamper.Socket) {
	c.client.AddSocket(sock)
}

func (c *plControllerT) getSocket(n string) (*scamper.Socket, error) {
	return c.client.GetSocket(n)
}

// Start starts a plcontroller with the given configuration
func Start(c Config, noScamp bool, db da.VantagePointProvider, cl Client) chan error {
	glog.Info("Starting plcontroller")
	errChan := make(chan error, 2)
	if db == nil {
		errChan <- fmt.Errorf("Nil db in plController")
		return errChan
	}
	plController.db = db
	var sc scamper.Config
	sc.Port = c.Scamper.Port
	sc.Path = c.Scamper.SockDir
	sc.ScPath = c.Scamper.BinPath
	sc.ScParserPath = c.Scamper.ConverterPath
	err := scamper.ParseConfig(sc)
	if err != nil {
		glog.Errorf("Invalid scamper args: %v", err)
		errChan <- err
		return errChan
	}
	plController.spoofs = newSpoofMap()
	plController.config = c
	plController.startTime = time.Now()
	plController.mp = mproc.New()
	plController.sc = sc
	plController.conf = c
	if !noScamp {
		plController.startScamperProc()
	}
	plController.client = cl
	//Watch dir doesn't make the scamper dir if it doesn't exist so it's
	//best to call it after startScamperProc otherwise you'll send an error
	//and trigger any error logic in whatever code is using this
	plController.watchDir(sc.Path, errChan)
	var opts []grpc.ServerOption
	plController.server = grpc.NewServer(opts...)
	plc.RegisterPLControllerServer(plController.server, &plController)
	go plController.startRPC(errChan)
	return errChan
}

func (c *plControllerT) startRPC(eChan chan error) {
	var addr string
	if c.config.Local.AutoConnect {
		saddr, err := util.GetBindAddr()
		if err != nil {
			eChan <- err
			return
		}
		addr = fmt.Sprintf("%s:%d", saddr, 45000)
	} else {
		addr = c.config.Local.Addr
	}
	glog.Infof("Conecting to: %s", addr)
	l, e := net.Listen(c.config.Local.Proto, addr)
	if e != nil {
		glog.Errorf("Failed to listen: %v", e)
		eChan <- e
		return
	}
	glog.Infof("PLController started, listening on: %s", addr)
	err := c.server.Serve(l)
	if err != nil {
		eChan <- err
	}
}

func (c *plControllerT) startScamperProc() {
	sp := scamper.GetProc(c.sc.Path, c.sc.Port, c.sc.ScPath)
	plController.mp.ManageProcess(sp, true, 1000, handleScamperStop)
}

// HandleSig allows the plController to react appropriately to signals
func HandleSig(s os.Signal) {
	plController.handleSig(s)
}

func (c *plControllerT) handleSig(s os.Signal) {
	glog.Infof("Got signal %v", s)
	if c.mp != nil {
		c.mp.KillAll()
	}
	if c.w != nil {
		c.w.Close()
	}
	c.removeAllVps()
	c.db.Close()
}
