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
package plcontroller

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
)

var (
	addr          = "localhost:45000"
	proto         = "tcp"
	port          = "45454"
	sockdir       = "/tmp/scamper_sockets"
	binpath       = "/usr/local/bin/sc_remoted"
	converterpath = "/usr/local/bin/sc_warts2json"
)

var conf = Config{
	Local: LocalConfig{
		Addr: &addr,
	},
	Scamper: ScamperConfig{
		Port:          &port,
		SockDir:       &sockdir,
		BinPath:       &binpath,
		ConverterPath: &converterpath,
	},
}

/*
func TestStart(t *testing.T) {
	vp, _ := testdataaccess.NewVP()
	eChan := Start(conf, true, vp, scamper.NewClient(), ControllerSender{})

	select {
	case e := <-eChan:
		t.Fatal("TestStart failed %v", e)
	case <-time.After(time.Second * 2):

	}

}

func TestStartInvalidIP(t *testing.T) {
	var (
		addr        = "-1:45000"
		port        = "45454"
		sockdir     = "/tmp/scamper_sockets"
		binpath     = "/usr/local/bin/sc_remoted"
		convertpath = "/usr/local/bin/sc_warts2json"
	)
	var c = Config{
		Local: LocalConfig{
			Addr: &addr,
		},
		Scamper: ScamperConfig{
			Port:          &port,
			SockDir:       &sockdir,
			BinPath:       &binpath,
			ConverterPath: &convertpath,
		},
	}
	vp, _ := testdataaccess.NewVP()
	eChan := Start(c, true, vp, scamper.NewClient(), ControllerSender{})
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartInvalidIP no error thrown with invalid ip")
	}

}

func TestStartInvalidPort(t *testing.T) {
	var (
		addr        = "localhost:PORT"
		port        = "45454"
		sockdir     = "/tmp/scamper_sockets"
		binpath     = "/usr/local/bin/sc_remoted"
		convertpath = "/usr/local/bin/sc_warts2json"
	)
	var c = Config{
		Local: LocalConfig{
			Addr: &addr,
		},
		Scamper: ScamperConfig{
			Port:          &port,
			SockDir:       &sockdir,
			BinPath:       &binpath,
			ConverterPath: &convertpath,
		},
	}
	vp, _ := testdataaccess.NewVP()
	eChan := Start(c, true, vp, scamper.NewClient(), ControllerSender{})
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartInvalidPort no error thrown with invalid port")
	}

}

func TestStartPortOutOfRange(t *testing.T) {
	var (
		addr        = "localhost:7000"
		port        = "45454"
		sockdir     = "/tmp/scamper_sockets"
		binpath     = "/usr/local/bin/sc_remoted"
		convertpath = "/usr/local/bin/sc_warts2json"
	)
	var c = Config{
		Local: LocalConfig{
			Addr: &addr,
		},
		Scamper: ScamperConfig{
			Port:          &port,
			SockDir:       &sockdir,
			BinPath:       &binpath,
			ConverterPath: &convertpath,
		},
	}

	vp, _ := testdataaccess.NewVP()
	eChan := Start(c, true, vp, scamper.NewClient(), ControllerSender{})
	select {
	case <-eChan:
	case <-time.After(time.Second * 2):
		t.Fatal("TestStartPortOutOfRange no error thrown with port 70000")
	}
}
*/
func TestInstallService(t *testing.T) {
	vp := &datamodel.VantagePoint{
		Hostname: "mlab1.prg01.measurement-lab.org",
		Port:     806,
	}
	random := rand.Int()
	err := installService(getCmd(vp, "uw_geoloc4", "/home/rhansen2/.ssh/id_rsa_pl", fmt.Sprintf(install, random, random, "http://www.ccs.neu.edu/home/rhansen2/plvp-0.0.1-2.i386.rpm", "plvp-0.0.1-2.i386.rpm")))
	if err != nil {
		t.Fatalf("Failed to install service: %v", err)
	}
}
