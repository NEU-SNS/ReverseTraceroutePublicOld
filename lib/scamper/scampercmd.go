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
	"bytes"
	"fmt"
	"io"
	"reflect"

	dm "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
	"github.com/golang/glog"
)

const (
	PING       cmdT = "ping"
	TRACEROUTE cmdT = "trace"
)

var optionMap = map[cmdT]map[string]option{
	"ping":  pingArg,
	"trace": traceArg,
}

type option struct {
	format string
	opt    OptGetter
}

func intOpt(f string, arg interface{}) (string, error) {
	if sarg, ok := arg.(int); ok {
		return fmt.Sprintf(f, sarg), nil
	}
	return "", fmt.Errorf("Invalid arg type in intOpt: %v", arg)
}

func boolOpt(f string, arg interface{}) (string, error) {
	if barg, ok := arg.(bool); ok {
		if barg {
			return f, nil
		} else {
			return "", nil
		}
	}
	return "", fmt.Errorf("Invalid arg type in boolOpt: %v", arg)
}

func stringOpt(f string, arg interface{}) (string, error) {
	if sarg, ok := arg.(string); ok {
		if sarg == "" {
			return sarg, nil
		}
		return fmt.Sprintf(f, sarg), nil
	}
	return "", fmt.Errorf("Invalid arg type in stringOpt: %v", arg)
}

type OptGetter func(f string, arg interface{}) (string, error)

type cmdT string

type Cmd struct {
	ct          cmdT
	options     []string
	id          uint32
	userIDCache uint32
	resp        Response
	userID      uint32
	arg         interface{}
}

func (c Cmd) Marshal() []byte {
	var buf bytes.Buffer
	glog.Infof("CMD: %s, %d", c.ct, len(string(c.ct)))
	buf.WriteString(string(c.ct) + " ")
	for _, arg := range c.options {
		buf.WriteString(arg + " ")
	}
	buf.WriteString("\n")
	glog.Infof("Cmd as string: %s", cmd)
	return buf.Bytes()
}

func (c *Cmd) IssueCommand(w io.Writer) error {
	cmd := c.Marshal()
	_, err := w.Write(cmd)
	return err
}

func newCmd(arg interface{}, id uint32) (c Cmd, err error) {
	switch arg.(type) {
	case dm.PingArg:
		oID := arg.(dm.PingArg).UserId
		arg.(dm.PingArg).UserId = id
		c, err = createCmd(arg, PING)
		c.userIDCache = oID
		c.userID = id
	case dm.TracerouteArg:
		oID := arg.(dm.TracerouteArg).Userid
		arg.(dm.TracerouteArg).Userid = id
		c, err = createCmd(arg, TRACEROUTE)
		c.userIDCache = oID
		c.userID = id
	default:
		err = fmt.Errorf("Failed to create Cmd, type not found")
	}
	c.arg = arg
	return
}

func createCmd(arg interface{}, t cmdT) (Cmd, error) {
	//This far from handles all possible cases
	opts := optionMap[t]
	ty := reflect.TypeOf(arg)
	v := reflect.ValueOf(arg)
	n := v.NumField()
	fopts := make([]string, n)
	var targ string
	for i := 0; i < n; i++ {
		f := ty.Field(i)
		if o, ok := opts[f.Name]; ok {
			str, err := o.opt(o.format, v.FieldByName(f.Name).Interface())
			if err != nil {
				glog.Errorf("Failed on option: %s", f.Name)
				return Cmd{}, fmt.Errorf("Error creating option err: %v", err)
			}
			if f.Name == "Dst" {
				targ = str
				continue
			}
			fopts = append(fopts, str)
		}
	}
	fopts = append(fopts, targ)
	return Cmd{ct: t, options: fopts}, nil
}
