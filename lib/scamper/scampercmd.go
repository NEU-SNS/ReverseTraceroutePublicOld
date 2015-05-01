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
package scamper

import (
	"fmt"
	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"reflect"
)

const (
	PING CmdT = "ping"
)

var optionMap = map[CmdT]map[string]option{
	"ping": map[string]option{
		"Dst": option{
			format: "%s ",
			opt:    stringOpt,
		},
		"Spoof": option{
			format: "-O spoof ",
			opt:    boolOpt,
		},
		"SAddr": option{
			format: "-S %s ",
			opt:    stringOpt,
		},
		"RR": option{
			format: "-RR ",
			opt:    boolOpt,
		},
	},
}

type option struct {
	format string
	opt    OptGetter
}

func boolOpt(f string, arg interface{}) (string, error) {
	if barg, ok := arg.(bool); ok && barg {
		return f, nil
	}
	return "", fmt.Errorf("Invalid arg type in boolOpt: %v", arg)
}

func stringOpt(f string, arg interface{}) (string, error) {
	if sarg, ok := arg.(string); ok {
		return fmt.Sprintf(f, sarg), nil
	}
	return "", fmt.Errorf("Invalid arg type in stringOpt: %v", arg)
}

type OptGetter func(f string, arg interface{}) (string, error)

type CmdT string

type Cmd struct {
	ct      CmdT
	options []string
}

func (c Cmd) String() string {
	return fmt.Sprintf("TODO")
}

func NewCmd(arg interface{}) (Cmd, error) {
	switch arg.(type) {
	case dm.PingArg:
		return createCmd(arg, PING)
	}
	return Cmd{}, fmt.Errorf("Failed to create Cmd, type not found")
}

func createCmd(arg interface{}, t CmdT) (Cmd, error) {
	//This far from handles all possible cases
	opts := optionMap[t]
	ty := reflect.TypeOf(opts)
	v := reflect.ValueOf(opts)
	n := v.NumField()
	fopts := make([]string, n)
	for i := 0; i < n; i++ {
		f := ty.Field(i)
		if o, ok := opts[f.Name]; ok {
			str, err := o.opt(o.format, v.FieldByName(f.Name).Interface())
			if err != nil {
				return Cmd{}, fmt.Errorf("Error creating option err: %v", err)
			}
			fopts = append(fopts, str)
		}
	}
	return Cmd{ct: t, options: fopts}, nil
}
