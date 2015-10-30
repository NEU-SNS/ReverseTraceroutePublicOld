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
package sql_test

import (
	"os"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/log"
)

var conf = sql.DbConfig{
	WriteConfigs: []sql.Config{
		sql.Config{
			User:     "revtr",
			Password: "password",
			Host:     "localhost",
			Port:     "3306",
			Db:       "ccontroller",
		},
	},
	ReadConfigs: []sql.Config{
		sql.Config{
			User:     "revtr",
			Password: "password",
			Host:     "localhost",
			Port:     "3306",
			Db:       "ccontroller",
		},
	},
}

var db *sql.DB

func TestMain(m *testing.M) {
	var err error
	db, err = sql.NewDB(conf)
	if err != nil {
		log.Info(err)
		os.Exit(1)
	}
	defer db.Close()
	result := m.Run()
	os.Exit(result)
}

func TestGetTracerouteBadIp(t *testing.T) {
	_, err := db.GetTRBySrcDst(0, 0)
	if err != nil {
		t.Fatalf("TestGetTracerouteBadIp: %v", err)
	}
}

func TestGetTraceroute(t *testing.T) {
	_, err := db.GetTRBySrcDst(89, 90)
	if err != nil {
		t.Fatalf("TestGetTracerouteBadIp: %v", err)
	}
}

func TestInsertTraceroute(t *testing.T) {
	var tt dm.TracerouteTime
	unix := time.Now().Unix()
	tt.Sec = int64(unix)
	test := dm.Traceroute{
		Type:       "test",
		Src:        89,
		Dst:        90,
		UserId:     111,
		Method:     "testing",
		Sport:      111,
		Dport:      222,
		StopReason: "test",
		StopData:   12,
		Start:      &tt,
		HopCount:   2,
		Attempts:   3,
		Hoplimit:   4,
		Firsthop:   1,
		Wait:       88,
		WaitProbe:  99,
		Tos:        6,
		Version:    "-1",
	}
	err := db.StoreTraceroute(&test)
	if err != nil {
		t.Fatalf("TestInsertTraceroute: %v", err)
	}

}

func TestInsertPing(t *testing.T) {
	var start dm.Time
	start.Sec = time.Now().Unix()
	ping := dm.Ping{
		Type:      "test",
		Src:       89,
		Dst:       90,
		UserId:    111,
		Start:     &start,
		PingSent:  4,
		ProbeSize: 5,
		Ttl:       90,
		Wait:      20,
		Timeout:   10,
		Flags:     []string{"v4rr"},
		Version:   "-1",
	}
	err := db.StorePing(&ping)
	if err != nil {
		t.Fatalf("TestInsertPing: %v", err)
	}
}

func TestGetPing(t *testing.T) {
	_, err := db.GetPingBySrcDst(89, 90)
	if err != nil {
		t.Fatalf("TestGetPing: %v", err)
	}
}

/*
func TestGetVps(t *testing.T) {
	vps, err := db.GetVPs()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(vps)

}

const (
	cip = 167772161
)

func TestUpdateControllerSetToNull(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 167772161, cip)
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpdateControllerNullController(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 0, cip)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateControllerDifferentController(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 167772162, cip)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateCanSpoof(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateCanSpoof(ip, true)
	if err != nil {
		t.Fatal(err)
	}
	err = db.UpdateCanSpoof(ip, false)
	if err != nil {
		t.Fatal(err)
	}
}
*/
