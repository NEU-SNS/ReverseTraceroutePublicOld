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
	"os"
	"testing"
)

type SubConfig struct {
	Name string
	Age  int
}

type Config struct {
	Name   string
	Num    int
	Array  []int
	SubCon SubConfig
}

const config = `
name: Rob
num: 65
array: [1, 2, 3]
subcon:
    name: Rob
    age: 25
`

var testConfig = Config{
	Name:  "Rob",
	Num:   65,
	Array: []int{1, 2, 3},
	SubCon: SubConfig{
		Name: "Rob",
		Age:  25,
	},
}

func TestParseConfig(t *testing.T) {
	f, err := os.Create("testparse.config")
	if err != nil {
		t.Fatalf("Error creating file: %v", err)
	}
	defer os.Remove("testparse.config")
	var conf Config
	f.Write([]byte(config))
	err = ParseConfig("testparse.config", &conf)
	t.Logf("%v", conf)
	t.Logf("%v", testConfig)
	if err != nil {
		t.Fatalf("Failed parse config: %v", err)
	}

	if testConfig.Name != conf.Name {
		t.Fatalf("Failed to parse Config.Name")
	}
	if testConfig.Num != conf.Num {
		t.Fatalf("Failed to parse Config.Num")
	}
	if testConfig.SubCon != conf.SubCon {
		t.Fatalf("Failed to parse Config.SubCon")
	}
	if len(testConfig.Array) != len(conf.Array) {
		t.Fatalf("Failed to parse Config.Array")
	}

	for i := 0; i < len(testConfig.Array); i++ {
		if testConfig.Array[i] != conf.Array[i] {
			t.Fatalf("Failed to parse Config.Array[%d]", i)
		}
	}
}

func TestParseConfigBadPath(t *testing.T) {
	var conf Config
	err := ParseConfig("-", &conf)
	if err == nil {
		t.Fatalf("TestParseConfigBadPath: Didnt return error with bad path")
	}
}
