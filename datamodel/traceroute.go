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

package datamodel

import (
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
)

type TTime time.Time

const TRACETIME = "2006-01-_2 15:04:05"

func (t *TTime) UnmarshalJSON(data []byte) (err error) {
	temp, err := time.Parse(`"`+TRACETIME+`"`, string(data))
	*t = TTime(temp)
	return
}

func (t TTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if y := tt.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("TTime.MarshalJSON: year outside of range [0,9999]")
	}
	return []byte(tt.Format(`"` + TRACETIME + `"`)), nil
}

func (t TTime) String() string {
	tt := time.Time(t)
	return tt.String()
}

func (t *Traceroute) Marshal() []byte {
	ret, err := proto.Marshal(t)
	if err != nil {
		return nil
	}
	return ret
}

func (t *Traceroute) Key() string {
	return ""
}
func (t *Traceroute) CUnmarshal(data []byte) error {
	return proto.Unmarshal(data, t)
}

func (t *Traceroute) CMarshal() []byte {
	ret, err := proto.Marshal(t)
	if err != nil {
		return nil
	}
	return ret
}

func (tm *TracerouteMeasurement) CMarshal() []byte {
	ret, err := proto.Marshal(tm)
	if err != nil {
		return nil
	}
	return ret
}

func (tm *TracerouteMeasurement) Key() string {
	return ""
}
