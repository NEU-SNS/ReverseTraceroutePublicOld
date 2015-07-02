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
package scamper_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/golang/glog"
)

var sockPath = "/tmp/192.168.1.2:5000"
var testIP = "192.168.1.2"
var testPort = "5000"

func TestMain(m *testing.M) {
	setupSocket()
	result := m.Run()
	cleanupSocket()
	// Clear out glog buffer
	glog.Flush()
	os.Exit(result)
}

var listener net.Listener

func setupSocket() {
	lis, _ := net.Listen("unix", sockPath)
	listener = lis
	go func() {
		for {
			con, err := lis.Accept()
			content, err := ioutil.ReadFile("../doc/test_scamper.txt")
			if err != nil {
				con.Close()
				continue
			}
			_, err = con.Write(content)
		}
	}()

}

func cleanupSocket() {
	os.Remove(sockPath)
}

func TestSocket(t *testing.T) {
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)

	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	soc.Stop()

}

func TestSocketDoMeasurement(t *testing.T) {
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)

	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	testIDStr := "0"
	var testID uint32
	ping := dm.PingArg{
		Service: dm.ServiceT_PLANET_LAB,
		Dst:     "8.8.8.8",
		Src:     testIP,
		UserId:  testIDStr,
	}

	rec, err := soc.DoMeasurement(ping)
	if err != nil {
		t.Fatalf("Failed to do measurement: %v", err)
	}
	select {
	case r := <-rec:
		if r.UserID != testID {
			t.Fatalf("UserId did not match %d != %d", testID, r.UserID)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Timeout running measurement")
	}
	soc.Stop()
}

func TestSocketIP(t *testing.T) {
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	ip := soc.IP()
	if ip != testIP {
		t.Fatalf("SocketIP failed, got: %s expected: %s", ip, testIP)
	}
}

func TestSocketPort(t *testing.T) {
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	port := soc.Port()
	if port != testPort {
		t.Fatalf("SocketIP failed, got: %s expected: %s", port, testPort)
	}
}

func TestClientDoMeasurementNoSocket(t *testing.T) {
	client := scamper.NewClient()
	_, err := client.DoMeasurement("192.168.1.1", 6)
	if err == nil {
		t.Fatal("Client failed to throw error for unknown socket")
	}
}

func TestClientAddSocket(t *testing.T) {
	client := scamper.NewClient()
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}

	client.AddSocket(soc)

}

func TestClientRemoveSocket(t *testing.T) {
	client := scamper.NewClient()
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}

	client.AddSocket(soc)
	client.RemoveSocket(soc.IP())
}

func TestClientGetSocket(t *testing.T) {
	client := scamper.NewClient()
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)
	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}

	client.AddSocket(soc)
	soc, err = client.GetSocket(soc.IP())
	if err != nil {
		t.Fatal("Failed to get registered socket")
	}
}

func TestClientDoMeasurement(t *testing.T) {
	soc, err := scamper.NewSocket(
		sockPath,
		"/usr/local/bin/sc_warts2json",
		json.Unmarshal,
		net.Dial)

	if err != nil {
		t.Fatalf("Failed to create socket: %v", err)
	}
	testIDStr := "0"
	var testID uint32
	ping := dm.PingArg{
		Service: dm.ServiceT_PLANET_LAB,
		Dst:     "8.8.8.8",
		Src:     testIP,
		UserId:  testIDStr,
	}
	client := scamper.NewClient()
	client.AddSocket(soc)
	rec, err := client.DoMeasurement(soc.IP(), ping)
	if err != nil {
		t.Fatalf("Failed to do measurement: %v", err)
	}
	select {
	case r := <-rec:
		if r.UserID != testID {
			t.Fatalf("UserId did not match %d != %d", testID, r.UserID)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Timeout running measurement")
	}
	soc.Stop()
}
