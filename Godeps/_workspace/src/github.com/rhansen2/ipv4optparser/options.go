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
package ipv4opt

import (
	"fmt"
	"net"
)

type OptionType uint8
type OptionLength uint8
type RouteAddress uint32
type OptionData uint8
type SecurityLevel uint16
type SecurityCompartment uint16
type SecurityHandlingRestriction uint16
type SecurityTCC uint32
type Route uint32
type StreamID uint16
type Timestamp uint32
type Flag uint8
type Overflow uint8
type Address uint32

func (addr Address) String() string {
	var a, b, c, d byte
	a = byte(addr >> 24)
	b = byte((addr & 0x00ff0000) >> 16)
	c = byte((addr & 0x0000ff00) >> 8)
	d = byte(addr & 0x000000ff)
	return net.IPv4(a, b, c, d).String()
}

func (r Route) String() string {
	var a, b, c, d byte
	a = byte(r >> 24)
	b = byte((r & 0x00ff0000) >> 16)
	c = byte((r & 0x0000ff00) >> 8)
	d = byte(r & 0x000000ff)
	return net.IPv4(a, b, c, d).String()
}

const (
	EndOfOptionList         OptionType = 0
	NoOperation                        = 1
	Security                           = 130
	LooseSourceRecordRoute             = 131
	StrictSourceRecordRoute            = 137
	RecordRoute                        = 7
	StreamIdentifier                   = 136
	InternetTimestamp                  = 68
	MaxOptionsLen           int        = 40 // 60 Byte maximum size - 20 bytes for manditory fields

	Unclassified SecurityLevel = 0x0
	Confidential               = 0xF135
	EFTO                       = 0x789A
	MMMM                       = 0xBC4D
	PROG                       = 0x5E26
	Restricted                 = 0xAF13
	Secret                     = 0xD788
	TopSecret                  = 0x6BC5
	Reserved0                  = 0x35E2
	Reserved1                  = 0x9AF1
	Reserved2                  = 0x4D78
	Reserved3                  = 0x24BD
	Reserved4                  = 0x135E
	Reserved5                  = 0x89AF
	Reserved6                  = 0xC4D6
	Reserved7                  = 0xE26B
)

const (
	TSOnly    = 0
	TSAndAddr = 1
	TSPrespec = 3
)

var (
	ErrorOptionDataTooLarge      = fmt.Errorf("The length of the options data is larger than the max options length")
	ErrorOptionType              = fmt.Errorf("Invalid option type")
	ErrorNegativeOptionLength    = fmt.Errorf("Negative option length")
	ErrorNotEnoughData           = fmt.Errorf("Not enough data left to parse option")
	ErrorOptionTypeMismatch      = fmt.Errorf("Tried to convert an option to the wrong type")
	ErrorInvalidLength           = fmt.Errorf("The option length is incorrect")
	ErrorRouteLengthIncorrect    = fmt.Errorf("The length of the route data is not a multiple of 4")
	ErrorTSLengthIncorrect       = fmt.Errorf("The length of the route data is not a multiple of 4")
	ErrorStreamIDLengthIncorrect = fmt.Errorf("Then stream ID length is not 4")
)

type Option struct {
	Type   OptionType
	Length OptionLength
	Data   []OptionData
}

type Options []Option

type SecurityOption struct {
	Type        OptionType
	Length      OptionLength
	Level       SecurityLevel
	Compartment SecurityCompartment
	Restriction SecurityHandlingRestriction
	TCC         SecurityTCC
}

func (o Option) ToSecurity() (SecurityOption, error) {
	so := SecurityOption{}
	so.Type = o.Type
	so.Length = o.Length
	if o.Type != Security {
		return so, ErrorOptionTypeMismatch
	}
	if o.Length != 11 {
		return so, ErrorInvalidLength
	}
	data := o.Data
	so.Level |= SecurityLevel(data[0]) << 8
	so.Level |= SecurityLevel(data[1])

	so.Compartment |= SecurityCompartment(data[2]) << 8
	so.Compartment |= SecurityCompartment(data[3])

	so.Restriction |= SecurityHandlingRestriction(data[4]) << 8
	so.Restriction |= SecurityHandlingRestriction(data[5])

	so.TCC |= SecurityTCC(data[6]) << 16
	so.TCC |= SecurityTCC(data[7]) << 8
	so.TCC |= SecurityTCC(data[8])

	return so, nil
}

type RecordRouteOption struct {
	Type   OptionType
	Length OptionLength
	Routes []Route
}

func (o Option) ToRecordRoute() (RecordRouteOption, error) {
	rro := RecordRouteOption{}
	rro.Type = o.Type
	rro.Length = o.Length
	if o.Type != StrictSourceRecordRoute &&
		o.Type != LooseSourceRecordRoute &&
		o.Type != RecordRoute {
		return rro, ErrorOptionTypeMismatch
	}
	routeLen := rro.Length - 3 // The length of routes is length - 3 because length include the pointer type and length
	if routeLen%4 != 0 {
		return rro, ErrorRouteLengthIncorrect
	}
	for i := 0; i < int(routeLen); i += 4 {
		var route Route
		route |= Route(o.Data[i]) << 24
		route |= Route(o.Data[i+1]) << 16
		route |= Route(o.Data[i+2]) << 8
		route |= Route(o.Data[i+3])

		rro.Routes = append(rro.Routes, route)
	}
	return rro, nil
}

type StreamIdentifierOption struct {
	Type   OptionType
	Length OptionLength
	ID     StreamID
}

func (o Option) ToStreamID() (StreamIdentifierOption, error) {
	sid := StreamIdentifierOption{}
	sid.Type = o.Type
	sid.Length = o.Length
	if o.Type != StreamIdentifier {
		return sid, ErrorOptionTypeMismatch
	}
	if o.Length != 4 {
		return sid, ErrorStreamIDLengthIncorrect
	}
	sid.ID |= StreamID(o.Data[0]) << 8
	sid.ID |= StreamID(o.Data[1])

	return sid, nil

}

type Stamp struct {
	Time Timestamp
	Addr Address
}

type TimeStampOption struct {
	Type   OptionType
	Length OptionLength
	Flags  Flag
	Over   Overflow
	Stamps []Stamp
}

func (o Option) ToTimeStamp() (TimeStampOption, error) {
	ts := TimeStampOption{}
	ts.Type = o.Type
	ts.Length = o.Length
	if o.Type != InternetTimestamp {
		return ts, ErrorOptionTypeMismatch
	}
	if len(o.Data) > MaxOptionsLen {
		return ts, ErrorOptionDataTooLarge
	}
	ts.Over = Overflow(o.Data[1] >> 4)
	ts.Flags = Flag(o.Data[1] & 0x0F)
	// Take off two because of the flag and overflow byte and the ponter byte
	if len(o.Data)%4-2 != 0 && ts.Flags != TSOnly {
		return ts, ErrorTSLengthIncorrect
	}
	var err error
	switch ts.Flags {
	case TSOnly:
		ts.Stamps, err = getStampsTSOnly(o.Data[2:], len(o.Data)-2)
		if err != nil {
			return ts, err
		}
	case TSAndAddr, TSPrespec:
		ts.Stamps, err = getStamps(o.Data[2:], len(o.Data)-2)
		if err != nil {
			return ts, err
		}
	}
	return ts, nil
}

func getStampsTSOnly(data []OptionData, length int) ([]Stamp, error) {
	stamp := make([]Stamp, 0)
	for i := 0; i < length; i += 4 {
		st := Stamp{}
		st.Time |= Timestamp(data[i]) << 24
		st.Time |= Timestamp(data[i+1]) << 16
		st.Time |= Timestamp(data[i+2]) << 8
		st.Time |= Timestamp(data[i+3])
		stamp = append(stamp, st)
	}
	return stamp, nil
}

func getStamps(data []OptionData, length int) ([]Stamp, error) {
	stamp := make([]Stamp, 0)
	for i := 0; i < length; i += 8 {
		st := Stamp{}
		st.Addr |= Address(data[i]) << 24
		st.Addr |= Address(data[i+1]) << 16
		st.Addr |= Address(data[i+2]) << 8
		st.Addr |= Address(data[i+3])
		st.Time |= Timestamp(data[i+4]) << 24
		st.Time |= Timestamp(data[i+5]) << 16
		st.Time |= Timestamp(data[i+6]) << 8
		st.Time |= Timestamp(data[i+7])
		stamp = append(stamp, st)
	}
	return stamp, nil
}

func Parse(opts []byte) (Options, error) {
	optsLen := len(opts)
	if optsLen > MaxOptionsLen {
		return Options{}, ErrorOptionDataTooLarge
	}
	if optsLen == 0 {
		return Options{}, nil
	}
	options := make(Options, 0)
	for i := 0; i < optsLen; {
		option := Option{}
		oType, err := getOptionType(opts[i])
		if err != nil {
			return options, err
		}
		i++
		option.Type = oType
		if oType == EndOfOptionList {
			return append(options, option), nil
		}
		if oType == NoOperation {
			options = append(options, option)
			continue
		}
		data, l, n, err := parseOption(opts[i:])
		if err != nil {
			return Options{}, err
		}
		i += n
		option.Length = l
		option.Data = data
		options = append(options, option)
	}
	return options, nil

}

func parseOption(opts []byte) ([]OptionData, OptionLength, int, error) {
	l := opts[0]
	if l < 0 {
		return []OptionData{}, 0, 0, ErrorNegativeOptionLength
	}
	ol := OptionLength(l)
	// Length includes the length byte and type byte so read l - 2 more bytes
	rem := int(l) - 2
	if rem > len(opts)-1 { // If the remaining data is longer than the length of the options data - 1 for length byte
		return []OptionData{}, 0, 0, ErrorNotEnoughData
	}
	// Add one to rem because the synax is [x:)
	dataBytes := opts[1 : rem+1]
	dbl := len(dataBytes)
	ods := make([]OptionData, 0)
	for i := 0; i < dbl; i++ {
		ods = append(ods, OptionData(dataBytes[i]))
	}
	return ods, ol, int(l), nil
}

func getOptionType(b byte) (OptionType, error) {
	switch OptionType(b) {
	case EndOfOptionList:
		return EndOfOptionList, nil
	case NoOperation:
		return NoOperation, nil
	case Security:
		return Security, nil
	case LooseSourceRecordRoute:
		return LooseSourceRecordRoute, nil
	case StrictSourceRecordRoute:
		return StrictSourceRecordRoute, nil
	case RecordRoute:
		return RecordRoute, nil
	case StreamIdentifier:
		return StreamIdentifier, nil
	case InternetTimestamp:
		return InternetTimestamp, nil
	default:
		//Just return EndOfOptionList to satisfy return
		return EndOfOptionList, ErrorOptionType
	}
}
