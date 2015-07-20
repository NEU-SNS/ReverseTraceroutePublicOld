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

	"github.com/NEU-SNS/ReverseTraceroute/dataaccess/sql"
	"github.com/golang/glog"
)

var conf = sql.DbConfig{
	UName:    "revtr",
	Password: "password",
	Host:     "localhost",
	Port:     "3306",
	Db:       "plcontroller",
}

var db *sql.DB

func TestMain(m *testing.M) {
	var err error
	db, err = sql.NewDB(conf)
	if err != nil {
		glog.Info(err)
		glog.Flush()
		os.Exit(1)
	}
	defer db.Close()
	result := m.Run()
	glog.Flush()
	os.Exit(result)
}

func TestGetVps(t *testing.T) {
	vps, err := db.GetVPs()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(vps)

}

func TestUpdateControllerNullController(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 0, 167772161)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateControllerDifferentController(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 167772162, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateControllerSetToNull(t *testing.T) {
	var ip uint32 = 68101001
	err := db.UpdateController(ip, 167772161, 0)
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
