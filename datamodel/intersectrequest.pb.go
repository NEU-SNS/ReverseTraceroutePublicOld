// Code generated by protoc-gen-go.
// source: github.com/NEU-SNS/ReverseTraceroute/datamodel/intersectrequest.proto
// DO NOT EDIT!

/*
Package datamodel is a generated protocol buffer package.

It is generated from these files:
	github.com/NEU-SNS/ReverseTraceroute/datamodel/intersectrequest.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/ping.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/recspoof.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/reverse_traceroute.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/service.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/time.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/traceroute.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/update.proto
	github.com/NEU-SNS/ReverseTraceroute/datamodel/vantagepoint.proto

It has these top-level messages:
	Hop
	Path
	IntersectionRequest
	IntersectionResponse
	TokenRequest
	TokenResponse
	PingMeasurement
	PingArg
	PingStats
	PingResponse
	TsAndAddr
	Ping
	RecSpoof
	Spoof
	SpoofedProbes
	SpoofedProbesResponse
	Probe
	RecordRoute
	TimeStamp
	Stamp
	NotifyRecSpoofResponse
	ReceiveSpoofedProbesResponse
	RevtrMeasurement
	RevtrRequest
	ReverseTraceroute
	RevtrHop
	RevtrResponse
	Time
	RTT
	TracerouteMeasurement
	TracerouteArg
	TracerouteHop
	Traceroute
	TracerouteTime
	UpdateResponse
	VantagePoint
	VPRequest
	VPReturn
	RRSpooferRequest
	RRSpooferResponse
	TSSpooferRequest
	TSSpooferResponse
*/
package datamodel

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type IResponseType int32

const (
	IResponseType_NONE_FOUND IResponseType = 0
	IResponseType_TOKEN      IResponseType = 1
	IResponseType_PATH       IResponseType = 2
	IResponseType_ERROR      IResponseType = 3
)

var IResponseType_name = map[int32]string{
	0: "NONE_FOUND",
	1: "TOKEN",
	2: "PATH",
	3: "ERROR",
}
var IResponseType_value = map[string]int32{
	"NONE_FOUND": 0,
	"TOKEN":      1,
	"PATH":       2,
	"ERROR":      3,
}

func (x IResponseType) String() string {
	return proto.EnumName(IResponseType_name, int32(x))
}
func (IResponseType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Hop struct {
	Ip  uint32 `protobuf:"varint,1,opt,name=Ip" json:"Ip,omitempty"`
	Ttl uint32 `protobuf:"varint,2,opt,name=ttl" json:"ttl,omitempty"`
}

func (m *Hop) Reset()                    { *m = Hop{} }
func (m *Hop) String() string            { return proto.CompactTextString(m) }
func (*Hop) ProtoMessage()               {}
func (*Hop) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Path struct {
	Address uint32 `protobuf:"varint,1,opt,name=address" json:"address,omitempty"`
	Dest    uint32 `protobuf:"varint,2,opt,name=dest" json:"dest,omitempty"`
	Hops    []*Hop `protobuf:"bytes,3,rep,name=hops" json:"hops,omitempty"`
}

func (m *Path) Reset()                    { *m = Path{} }
func (m *Path) String() string            { return proto.CompactTextString(m) }
func (*Path) ProtoMessage()               {}
func (*Path) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Path) GetHops() []*Hop {
	if m != nil {
		return m.Hops
	}
	return nil
}

type IntersectionRequest struct {
	Address      uint32 `protobuf:"varint,1,opt,name=address" json:"address,omitempty"`
	Dest         uint32 `protobuf:"varint,2,opt,name=dest" json:"dest,omitempty"`
	Staleness    int64  `protobuf:"varint,3,opt,name=staleness" json:"staleness,omitempty"`
	UseAliases   bool   `protobuf:"varint,4,opt,name=use_aliases" json:"use_aliases,omitempty"`
	IgnoreSource bool   `protobuf:"varint,5,opt,name=ignore_source" json:"ignore_source,omitempty"`
	Src          uint32 `protobuf:"varint,6,opt,name=src" json:"src,omitempty"`
}

func (m *IntersectionRequest) Reset()                    { *m = IntersectionRequest{} }
func (m *IntersectionRequest) String() string            { return proto.CompactTextString(m) }
func (*IntersectionRequest) ProtoMessage()               {}
func (*IntersectionRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type IntersectionResponse struct {
	Type  IResponseType `protobuf:"varint,1,opt,name=type,enum=datamodel.IResponseType" json:"type,omitempty"`
	Token uint32        `protobuf:"varint,2,opt,name=token" json:"token,omitempty"`
	Path  *Path         `protobuf:"bytes,3,opt,name=path" json:"path,omitempty"`
	Error string        `protobuf:"bytes,4,opt,name=error" json:"error,omitempty"`
}

func (m *IntersectionResponse) Reset()                    { *m = IntersectionResponse{} }
func (m *IntersectionResponse) String() string            { return proto.CompactTextString(m) }
func (*IntersectionResponse) ProtoMessage()               {}
func (*IntersectionResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *IntersectionResponse) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

type TokenRequest struct {
	Token uint32 `protobuf:"varint,1,opt,name=token" json:"token,omitempty"`
}

func (m *TokenRequest) Reset()                    { *m = TokenRequest{} }
func (m *TokenRequest) String() string            { return proto.CompactTextString(m) }
func (*TokenRequest) ProtoMessage()               {}
func (*TokenRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

type TokenResponse struct {
	Token uint32        `protobuf:"varint,1,opt,name=token" json:"token,omitempty"`
	Type  IResponseType `protobuf:"varint,2,opt,name=type,enum=datamodel.IResponseType" json:"type,omitempty"`
	Path  *Path         `protobuf:"bytes,3,opt,name=path" json:"path,omitempty"`
	Error string        `protobuf:"bytes,4,opt,name=error" json:"error,omitempty"`
}

func (m *TokenResponse) Reset()                    { *m = TokenResponse{} }
func (m *TokenResponse) String() string            { return proto.CompactTextString(m) }
func (*TokenResponse) ProtoMessage()               {}
func (*TokenResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *TokenResponse) GetPath() *Path {
	if m != nil {
		return m.Path
	}
	return nil
}

func init() {
	proto.RegisterType((*Hop)(nil), "datamodel.Hop")
	proto.RegisterType((*Path)(nil), "datamodel.Path")
	proto.RegisterType((*IntersectionRequest)(nil), "datamodel.IntersectionRequest")
	proto.RegisterType((*IntersectionResponse)(nil), "datamodel.IntersectionResponse")
	proto.RegisterType((*TokenRequest)(nil), "datamodel.TokenRequest")
	proto.RegisterType((*TokenResponse)(nil), "datamodel.TokenResponse")
	proto.RegisterEnum("datamodel.IResponseType", IResponseType_name, IResponseType_value)
}

var fileDescriptor0 = []byte{
	// 397 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x94, 0x92, 0x4f, 0x6f, 0xd3, 0x40,
	0x10, 0xc5, 0xf1, 0x9f, 0x94, 0x7a, 0x5c, 0xa7, 0x66, 0x0b, 0x92, 0x0f, 0x14, 0x55, 0x3e, 0xa0,
	0x0a, 0x09, 0x5b, 0x0a, 0x1f, 0x00, 0xf1, 0xc7, 0xa8, 0x11, 0x92, 0x8d, 0x5c, 0xf7, 0xc2, 0x25,
	0xda, 0xda, 0xa3, 0xc6, 0xc2, 0xf1, 0x9a, 0xdd, 0x35, 0x12, 0xe2, 0xc2, 0x47, 0x67, 0xbd, 0x71,
	0x4c, 0x72, 0x8a, 0x7a, 0xdc, 0x99, 0x37, 0x6f, 0x7f, 0x6f, 0x34, 0x90, 0x3c, 0xd4, 0x72, 0xdd,
	0xdf, 0x47, 0x25, 0xdb, 0xc4, 0x69, 0x72, 0xf7, 0xf6, 0x36, 0xbd, 0x8d, 0x73, 0xfc, 0x85, 0x5c,
	0x60, 0xc1, 0x69, 0x89, 0x9c, 0xf5, 0x12, 0xe3, 0x8a, 0x4a, 0xba, 0x61, 0x15, 0x36, 0x71, 0xdd,
	0xca, 0xa1, 0x57, 0x4a, 0x8e, 0x3f, 0x7b, 0x14, 0x32, 0xea, 0x38, 0x93, 0x8c, 0x38, 0x93, 0x22,
	0x7c, 0x05, 0xd6, 0x0d, 0xeb, 0x08, 0x80, 0xb9, 0xec, 0x02, 0xe3, 0xca, 0xb8, 0xf6, 0x88, 0x0b,
	0x96, 0x94, 0x4d, 0x60, 0x0e, 0x8f, 0xf0, 0x13, 0xd8, 0xdf, 0xa8, 0x5c, 0x93, 0x73, 0x78, 0x4a,
	0xab, 0x8a, 0xa3, 0x10, 0xa3, 0xea, 0x0c, 0xec, 0x4a, 0x39, 0x6e, 0x65, 0xe4, 0x25, 0xd8, 0x6b,
	0xd6, 0x89, 0xc0, 0xba, 0xb2, 0xae, 0xdd, 0xc5, 0x3c, 0x9a, 0x3e, 0x88, 0x94, 0x7b, 0xf8, 0xd7,
	0x80, 0x8b, 0xe5, 0x0e, 0xa5, 0x66, 0x6d, 0xbe, 0xa5, 0x39, 0x66, 0xfa, 0x0c, 0x1c, 0x21, 0x69,
	0x83, 0xed, 0x20, 0xb0, 0x54, 0xc9, 0x22, 0x17, 0xe0, 0xf6, 0x02, 0x57, 0xb4, 0xa9, 0xa9, 0x40,
	0x11, 0xd8, 0xaa, 0x78, 0x4a, 0x5e, 0x80, 0x57, 0x3f, 0xb4, 0x8c, 0xe3, 0x4a, 0xb0, 0x9e, 0x97,
	0x18, 0xcc, 0x74, 0x59, 0xe5, 0x10, 0xbc, 0x0c, 0x4e, 0x74, 0x8e, 0x3f, 0xf0, 0xfc, 0x90, 0x40,
	0x74, 0xac, 0x15, 0x48, 0x5e, 0x83, 0x2d, 0x7f, 0x77, 0xa8, 0xff, 0x9f, 0x2f, 0x82, 0x3d, 0xf0,
	0xe5, 0x4e, 0x53, 0xa8, 0x3e, 0xf1, 0x60, 0x26, 0xd9, 0x0f, 0x6c, 0x47, 0xb4, 0x4b, 0xb0, 0x3b,
	0xb5, 0x16, 0x4d, 0xe5, 0x2e, 0xce, 0xf7, 0xc6, 0xf4, 0xb6, 0x94, 0x1a, 0x39, 0x67, 0x5c, 0x03,
	0x3a, 0xe1, 0x25, 0x9c, 0x15, 0xc3, 0xf0, 0x2e, 0xf7, 0x64, 0xa6, 0x53, 0x87, 0x12, 0xbc, 0xb1,
	0x3d, 0x42, 0x1d, 0xf6, 0x27, 0x46, 0xf3, 0x08, 0xe3, 0xa3, 0xa0, 0xde, 0xbc, 0x07, 0xef, 0x70,
	0x7c, 0x0e, 0x90, 0x66, 0x69, 0xb2, 0xfa, 0x92, 0xdd, 0xa5, 0x9f, 0xfd, 0x27, 0xc4, 0x81, 0x59,
	0x91, 0x7d, 0x4d, 0x52, 0xdf, 0x20, 0xa7, 0xea, 0x0a, 0x3e, 0x14, 0x37, 0xbe, 0x39, 0x14, 0x93,
	0x3c, 0xcf, 0x72, 0xdf, 0xfa, 0xe8, 0x7e, 0xff, 0x7f, 0x47, 0xf7, 0x27, 0xfa, 0xb2, 0xde, 0xfd,
	0x0b, 0x00, 0x00, 0xff, 0xff, 0xd3, 0x58, 0xbe, 0xf8, 0xa2, 0x02, 0x00, 0x00,
}
