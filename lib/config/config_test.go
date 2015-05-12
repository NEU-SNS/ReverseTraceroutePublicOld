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
package config

import (
	"github.com/NEU-SNS/ReverseTraceroute/lib/plvp"
	"os"
	"testing"
)

const config = `
local:
    addr: 127.0.0.1:55000
    proto: tcp
    closestddesc: true
    pprofaddr: 127.0.0.1:55550
scamper:
    addrs: ['10.0.0.1:80', '10.0.0.2:80']
    binpath: /path/to/bin
    addr: 192.168.1.2:56
`

var testConfig = plvp.Config{Local: plvp.LocalConfig{
	Addr:         "127.0.0.1:55000",
	Proto:        "tcp",
	CloseStdDesc: true,
	PProfAddr:    "127.0.0.1:55550",
}, Scamper: plvp.ScamperConfig{Addrs: []string{"10.0.0.1:80", "10.0.0.2:80"},
	BinPath: "/path/to/bin", Addr: "192.168.1.2:56"}}

func TestParseConfig(t *testing.T) {
	f, err := os.Create("testparse.config")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}
	defer os.Remove("testparse.config")
	var conf plvp.Config
	f.Write([]byte(config))
	err = ParseConfig("testparse.config", &conf)
	t.Logf("%v", conf)
	t.Logf("%v", testConfig)
	if err != nil {
		t.Fatalf("Failed parse config: %v", err)
	}

	if testConfig.Local.Addr != conf.Local.Addr {
		t.Fatalf("Local.Addr not equal %s != %s",
			testConfig.Local.Addr, conf.Local.Addr)
	}
	if testConfig.Local.Proto != conf.Local.Proto {
		t.Fatalf("Local.Proto not equal %s != %s",
			testConfig.Local.Proto, conf.Local.Proto)
	}
	if testConfig.Local.CloseStdDesc != conf.Local.CloseStdDesc {
		t.Fatalf("Local.CloseStdDesc not equal %s != %s",
			testConfig.Local.CloseStdDesc, conf.Local.CloseStdDesc)
	}
	if testConfig.Local.PProfAddr != conf.Local.PProfAddr {
		t.Fatalf("Local.Addr not equal %s != %s",
			testConfig.Local.PProfAddr, conf.Local.PProfAddr)
	}
	if testConfig.Scamper.BinPath != conf.Scamper.BinPath {
		t.Fatalf("Scamper.BinPath not equal %s != %s",
			testConfig.Scamper.BinPath, conf.Scamper.BinPath)
	}
	if testConfig.Scamper.Addr != conf.Scamper.Addr {
		t.Fatalf("Scamper.Addr not equal %s != %s",
			testConfig.Scamper.Addr, conf.Scamper.Addr)
	}
	lenEqual := len(testConfig.Scamper.Addrs) == len(conf.Scamper.Addrs)
	if !lenEqual {
		t.Fatalf("Scamper.Addrs not the same length")
	}
	for i := 0; i < len(testConfig.Scamper.Addrs); i++ {
		if testConfig.Scamper.Addrs[i] != conf.Scamper.Addrs[i] {
			t.Fatalf("Scamper.Addrs[%d] not equal %s != %s", i,
				testConfig.Scamper.Addrs[i], conf.Scamper.Addrs[i])
		}
	}
}

func TestParseConfigBadPath(t *testing.T) {
	var conf plvp.Config
	err := ParseConfig("-", &conf)
	if err == nil {
		t.Fatalf("TestParseConfigBadPath: Didnt return error with bad path")
	}
}
