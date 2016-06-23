// Code generated by protoc-gen-go.
// source: github.com/NEU-SNS/ReverseTraceroute/revtr/pb/revtr.proto
// DO NOT EDIT!

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	github.com/NEU-SNS/ReverseTraceroute/revtr/pb/revtr.proto

It has these top-level messages:
	RevtrMeasurement
	RunRevtrReq
	RunRevtrResp
	GetRevtrReq
	GetRevtrResp
	GetSourcesReq
	GetSourcesResp
	Source
	ReverseTraceroute
	RevtrHop
	RevtrUser
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gengo/grpc-gateway/third_party/googleapis/google/api"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type RevtrHopType int32

const (
	RevtrHopType_DUMMY                                         RevtrHopType = 0
	RevtrHopType_DST_REV_SEGMENT                               RevtrHopType = 1
	RevtrHopType_DST_SYM_REV_SEGMENT                           RevtrHopType = 2
	RevtrHopType_TR_TO_SRC_REV_SEGMENT                         RevtrHopType = 3
	RevtrHopType_RR_REV_SEGMENT                                RevtrHopType = 4
	RevtrHopType_SPOOF_RR_REV_SEGMENT                          RevtrHopType = 5
	RevtrHopType_TS_ADJ_REV_SEGMENT                            RevtrHopType = 6
	RevtrHopType_SPOOF_TS_ADJ_REV_SEGMENT                      RevtrHopType = 7
	RevtrHopType_SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO              RevtrHopType = 8
	RevtrHopType_SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO_DOUBLE_STAMP RevtrHopType = 9
)

var RevtrHopType_name = map[int32]string{
	0: "DUMMY",
	1: "DST_REV_SEGMENT",
	2: "DST_SYM_REV_SEGMENT",
	3: "TR_TO_SRC_REV_SEGMENT",
	4: "RR_REV_SEGMENT",
	5: "SPOOF_RR_REV_SEGMENT",
	6: "TS_ADJ_REV_SEGMENT",
	7: "SPOOF_TS_ADJ_REV_SEGMENT",
	8: "SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO",
	9: "SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO_DOUBLE_STAMP",
}
var RevtrHopType_value = map[string]int32{
	"DUMMY":                                         0,
	"DST_REV_SEGMENT":                               1,
	"DST_SYM_REV_SEGMENT":                           2,
	"TR_TO_SRC_REV_SEGMENT":                         3,
	"RR_REV_SEGMENT":                                4,
	"SPOOF_RR_REV_SEGMENT":                          5,
	"TS_ADJ_REV_SEGMENT":                            6,
	"SPOOF_TS_ADJ_REV_SEGMENT":                      7,
	"SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO":              8,
	"SPOOF_TS_ADJ_REV_SEGMENT_TS_ZERO_DOUBLE_STAMP": 9,
}

func (x RevtrHopType) String() string {
	return proto.EnumName(RevtrHopType_name, int32(x))
}
func (RevtrHopType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type RevtrStatus int32

const (
	RevtrStatus_DUMMY_X   RevtrStatus = 0
	RevtrStatus_RUNNING   RevtrStatus = 1
	RevtrStatus_COMPLETED RevtrStatus = 2
	RevtrStatus_CANCELED  RevtrStatus = 3
)

var RevtrStatus_name = map[int32]string{
	0: "DUMMY_X",
	1: "RUNNING",
	2: "COMPLETED",
	3: "CANCELED",
}
var RevtrStatus_value = map[string]int32{
	"DUMMY_X":   0,
	"RUNNING":   1,
	"COMPLETED": 2,
	"CANCELED":  3,
}

func (x RevtrStatus) String() string {
	return proto.EnumName(RevtrStatus_name, int32(x))
}
func (RevtrStatus) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type RevtrMeasurement struct {
	Src            string `protobuf:"bytes,1,opt,name=src" json:"src,omitempty"`
	Dst            string `protobuf:"bytes,2,opt,name=dst" json:"dst,omitempty"`
	Staleness      uint32 `protobuf:"varint,3,opt,name=staleness" json:"staleness,omitempty"`
	Id             uint32 `protobuf:"varint,4,opt,name=id" json:"id,omitempty"`
	BackoffEndhost bool   `protobuf:"varint,5,opt,name=backoff_endhost" json:"backoff_endhost,omitempty"`
}

func (m *RevtrMeasurement) Reset()                    { *m = RevtrMeasurement{} }
func (m *RevtrMeasurement) String() string            { return proto.CompactTextString(m) }
func (*RevtrMeasurement) ProtoMessage()               {}
func (*RevtrMeasurement) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type RunRevtrReq struct {
	Revtrs []*RevtrMeasurement `protobuf:"bytes,1,rep,name=revtrs" json:"revtrs,omitempty"`
	Auth   string              `protobuf:"bytes,2,opt,name=auth" json:"auth,omitempty"`
}

func (m *RunRevtrReq) Reset()                    { *m = RunRevtrReq{} }
func (m *RunRevtrReq) String() string            { return proto.CompactTextString(m) }
func (*RunRevtrReq) ProtoMessage()               {}
func (*RunRevtrReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *RunRevtrReq) GetRevtrs() []*RevtrMeasurement {
	if m != nil {
		return m.Revtrs
	}
	return nil
}

type RunRevtrResp struct {
	BatchId uint32 `protobuf:"varint,1,opt,name=batch_id" json:"batch_id,omitempty"`
}

func (m *RunRevtrResp) Reset()                    { *m = RunRevtrResp{} }
func (m *RunRevtrResp) String() string            { return proto.CompactTextString(m) }
func (*RunRevtrResp) ProtoMessage()               {}
func (*RunRevtrResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type GetRevtrReq struct {
	BatchId uint32 `protobuf:"varint,1,opt,name=batch_id" json:"batch_id,omitempty"`
	Auth    string `protobuf:"bytes,2,opt,name=auth" json:"auth,omitempty"`
}

func (m *GetRevtrReq) Reset()                    { *m = GetRevtrReq{} }
func (m *GetRevtrReq) String() string            { return proto.CompactTextString(m) }
func (*GetRevtrReq) ProtoMessage()               {}
func (*GetRevtrReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

type GetRevtrResp struct {
	Revtrs []*ReverseTraceroute `protobuf:"bytes,1,rep,name=revtrs" json:"revtrs,omitempty"`
}

func (m *GetRevtrResp) Reset()                    { *m = GetRevtrResp{} }
func (m *GetRevtrResp) String() string            { return proto.CompactTextString(m) }
func (*GetRevtrResp) ProtoMessage()               {}
func (*GetRevtrResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *GetRevtrResp) GetRevtrs() []*ReverseTraceroute {
	if m != nil {
		return m.Revtrs
	}
	return nil
}

type GetSourcesReq struct {
	Auth string `protobuf:"bytes,1,opt,name=auth" json:"auth,omitempty"`
}

func (m *GetSourcesReq) Reset()                    { *m = GetSourcesReq{} }
func (m *GetSourcesReq) String() string            { return proto.CompactTextString(m) }
func (*GetSourcesReq) ProtoMessage()               {}
func (*GetSourcesReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

type GetSourcesResp struct {
	Srcs []*Source `protobuf:"bytes,1,rep,name=srcs" json:"srcs,omitempty"`
}

func (m *GetSourcesResp) Reset()                    { *m = GetSourcesResp{} }
func (m *GetSourcesResp) String() string            { return proto.CompactTextString(m) }
func (*GetSourcesResp) ProtoMessage()               {}
func (*GetSourcesResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *GetSourcesResp) GetSrcs() []*Source {
	if m != nil {
		return m.Srcs
	}
	return nil
}

type Source struct {
	Hostname string `protobuf:"bytes,1,opt,name=hostname" json:"hostname,omitempty"`
	Ip       string `protobuf:"bytes,2,opt,name=ip" json:"ip,omitempty"`
}

func (m *Source) Reset()                    { *m = Source{} }
func (m *Source) String() string            { return proto.CompactTextString(m) }
func (*Source) ProtoMessage()               {}
func (*Source) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

type ReverseTraceroute struct {
	Status     RevtrStatus `protobuf:"varint,1,opt,name=status,enum=pb.RevtrStatus" json:"status,omitempty"`
	Src        string      `protobuf:"bytes,2,opt,name=src" json:"src,omitempty"`
	Dst        string      `protobuf:"bytes,3,opt,name=dst" json:"dst,omitempty"`
	Runtime    int64       `protobuf:"varint,4,opt,name=runtime" json:"runtime,omitempty"`
	RrIssued   int32       `protobuf:"varint,5,opt,name=rr_issued" json:"rr_issued,omitempty"`
	TsIssued   int32       `protobuf:"varint,6,opt,name=ts_issued" json:"ts_issued,omitempty"`
	StopReason string      `protobuf:"bytes,7,opt,name=stop_reason" json:"stop_reason,omitempty"`
	Date       string      `protobuf:"bytes,8,opt,name=date" json:"date,omitempty"`
	Path       []*RevtrHop `protobuf:"bytes,9,rep,name=path" json:"path,omitempty"`
	Id         uint32      `protobuf:"varint,10,opt,name=id" json:"id,omitempty"`
	FailReason string      `protobuf:"bytes,11,opt,name=fail_reason" json:"fail_reason,omitempty"`
}

func (m *ReverseTraceroute) Reset()                    { *m = ReverseTraceroute{} }
func (m *ReverseTraceroute) String() string            { return proto.CompactTextString(m) }
func (*ReverseTraceroute) ProtoMessage()               {}
func (*ReverseTraceroute) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *ReverseTraceroute) GetPath() []*RevtrHop {
	if m != nil {
		return m.Path
	}
	return nil
}

type RevtrHop struct {
	Hop  string       `protobuf:"bytes,1,opt,name=hop" json:"hop,omitempty"`
	Type RevtrHopType `protobuf:"varint,2,opt,name=type,enum=pb.RevtrHopType" json:"type,omitempty"`
}

func (m *RevtrHop) Reset()                    { *m = RevtrHop{} }
func (m *RevtrHop) String() string            { return proto.CompactTextString(m) }
func (*RevtrHop) ProtoMessage()               {}
func (*RevtrHop) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

type RevtrUser struct {
	Id    uint32 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	Name  string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Email string `protobuf:"bytes,3,opt,name=email" json:"email,omitempty"`
	Max   uint32 `protobuf:"varint,4,opt,name=max" json:"max,omitempty"`
	Delay uint32 `protobuf:"varint,5,opt,name=delay" json:"delay,omitempty"`
	Key   string `protobuf:"bytes,6,opt,name=key" json:"key,omitempty"`
}

func (m *RevtrUser) Reset()                    { *m = RevtrUser{} }
func (m *RevtrUser) String() string            { return proto.CompactTextString(m) }
func (*RevtrUser) ProtoMessage()               {}
func (*RevtrUser) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func init() {
	proto.RegisterType((*RevtrMeasurement)(nil), "pb.RevtrMeasurement")
	proto.RegisterType((*RunRevtrReq)(nil), "pb.RunRevtrReq")
	proto.RegisterType((*RunRevtrResp)(nil), "pb.RunRevtrResp")
	proto.RegisterType((*GetRevtrReq)(nil), "pb.GetRevtrReq")
	proto.RegisterType((*GetRevtrResp)(nil), "pb.GetRevtrResp")
	proto.RegisterType((*GetSourcesReq)(nil), "pb.GetSourcesReq")
	proto.RegisterType((*GetSourcesResp)(nil), "pb.GetSourcesResp")
	proto.RegisterType((*Source)(nil), "pb.Source")
	proto.RegisterType((*ReverseTraceroute)(nil), "pb.ReverseTraceroute")
	proto.RegisterType((*RevtrHop)(nil), "pb.RevtrHop")
	proto.RegisterType((*RevtrUser)(nil), "pb.RevtrUser")
	proto.RegisterEnum("pb.RevtrHopType", RevtrHopType_name, RevtrHopType_value)
	proto.RegisterEnum("pb.RevtrStatus", RevtrStatus_name, RevtrStatus_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion3

// Client API for Revtr service

type RevtrClient interface {
	RunRevtr(ctx context.Context, in *RunRevtrReq, opts ...grpc.CallOption) (*RunRevtrResp, error)
	GetRevtr(ctx context.Context, in *GetRevtrReq, opts ...grpc.CallOption) (*GetRevtrResp, error)
	GetSources(ctx context.Context, in *GetSourcesReq, opts ...grpc.CallOption) (*GetSourcesResp, error)
}

type revtrClient struct {
	cc *grpc.ClientConn
}

func NewRevtrClient(cc *grpc.ClientConn) RevtrClient {
	return &revtrClient{cc}
}

func (c *revtrClient) RunRevtr(ctx context.Context, in *RunRevtrReq, opts ...grpc.CallOption) (*RunRevtrResp, error) {
	out := new(RunRevtrResp)
	err := grpc.Invoke(ctx, "/pb.Revtr/RunRevtr", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *revtrClient) GetRevtr(ctx context.Context, in *GetRevtrReq, opts ...grpc.CallOption) (*GetRevtrResp, error) {
	out := new(GetRevtrResp)
	err := grpc.Invoke(ctx, "/pb.Revtr/GetRevtr", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *revtrClient) GetSources(ctx context.Context, in *GetSourcesReq, opts ...grpc.CallOption) (*GetSourcesResp, error) {
	out := new(GetSourcesResp)
	err := grpc.Invoke(ctx, "/pb.Revtr/GetSources", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Revtr service

type RevtrServer interface {
	RunRevtr(context.Context, *RunRevtrReq) (*RunRevtrResp, error)
	GetRevtr(context.Context, *GetRevtrReq) (*GetRevtrResp, error)
	GetSources(context.Context, *GetSourcesReq) (*GetSourcesResp, error)
}

func RegisterRevtrServer(s *grpc.Server, srv RevtrServer) {
	s.RegisterService(&_Revtr_serviceDesc, srv)
}

func _Revtr_RunRevtr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunRevtrReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RevtrServer).RunRevtr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Revtr/RunRevtr",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RevtrServer).RunRevtr(ctx, req.(*RunRevtrReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Revtr_GetRevtr_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRevtrReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RevtrServer).GetRevtr(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Revtr/GetRevtr",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RevtrServer).GetRevtr(ctx, req.(*GetRevtrReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Revtr_GetSources_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSourcesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RevtrServer).GetSources(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pb.Revtr/GetSources",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RevtrServer).GetSources(ctx, req.(*GetSourcesReq))
	}
	return interceptor(ctx, in, info, handler)
}

var _Revtr_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pb.Revtr",
	HandlerType: (*RevtrServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RunRevtr",
			Handler:    _Revtr_RunRevtr_Handler,
		},
		{
			MethodName: "GetRevtr",
			Handler:    _Revtr_GetRevtr_Handler,
		},
		{
			MethodName: "GetSources",
			Handler:    _Revtr_GetSources_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: fileDescriptor0,
}

func init() {
	proto.RegisterFile("github.com/NEU-SNS/ReverseTraceroute/revtr/pb/revtr.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 812 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x54, 0xdd, 0x6e, 0xeb, 0x44,
	0x10, 0x26, 0xff, 0xf1, 0xe4, 0xcf, 0xd9, 0x73, 0x0e, 0xc7, 0x44, 0x07, 0x88, 0x2c, 0x40, 0x55,
	0xa4, 0x26, 0x22, 0x08, 0x21, 0xb8, 0x4b, 0x13, 0x13, 0x40, 0xf9, 0xa9, 0x6c, 0x07, 0xd1, 0x4a,
	0xc8, 0x72, 0x92, 0x6d, 0x13, 0x35, 0xb1, 0x8d, 0x77, 0x5d, 0x51, 0x21, 0x6e, 0x78, 0x05, 0x1e,
	0x80, 0x87, 0xe2, 0x15, 0xe0, 0x9a, 0x57, 0x60, 0xbc, 0xb6, 0x93, 0x38, 0x05, 0x9d, 0xbb, 0x9d,
	0x6f, 0x76, 0xbf, 0x99, 0xf9, 0x66, 0x76, 0xe0, 0xcb, 0xfb, 0x2d, 0xdf, 0x04, 0xcb, 0xee, 0xca,
	0xdd, 0xf7, 0x66, 0xda, 0xe2, 0xd2, 0x98, 0x19, 0x3d, 0x9d, 0x3e, 0x52, 0x9f, 0x51, 0xd3, 0xb7,
	0x57, 0xd4, 0x77, 0x03, 0x4e, 0x7b, 0x3e, 0x7d, 0xe4, 0x7e, 0xcf, 0x5b, 0x46, 0x87, 0xae, 0xe7,
	0xbb, 0xdc, 0x25, 0x59, 0x6f, 0xd9, 0x7a, 0x73, 0xef, 0xba, 0xf7, 0x3b, 0xda, 0xb3, 0xbd, 0x6d,
	0xcf, 0x76, 0x1c, 0x97, 0xdb, 0x7c, 0xeb, 0x3a, 0x2c, 0xba, 0xa1, 0xae, 0x41, 0xd6, 0xc3, 0x07,
	0x53, 0x6a, 0xb3, 0xc0, 0xa7, 0x7b, 0xea, 0x70, 0x52, 0x81, 0x1c, 0xf3, 0x57, 0x4a, 0xa6, 0x9d,
	0xb9, 0x90, 0x42, 0x63, 0xcd, 0xb8, 0x92, 0x15, 0x46, 0x13, 0x24, 0xc6, 0xed, 0x1d, 0x75, 0x28,
	0x63, 0x4a, 0x0e, 0xa1, 0x1a, 0x01, 0xc8, 0x6e, 0xd7, 0x4a, 0x5e, 0x9c, 0x5f, 0x43, 0x63, 0x69,
	0xaf, 0x1e, 0xdc, 0xbb, 0x3b, 0x8b, 0x3a, 0xeb, 0x8d, 0x8b, 0xef, 0x0a, 0xe8, 0x28, 0xab, 0x03,
	0xa8, 0xe8, 0x81, 0x23, 0x02, 0xe9, 0xf4, 0x27, 0xf2, 0x11, 0x14, 0x45, 0x96, 0x0c, 0x63, 0xe4,
	0x2e, 0x2a, 0xfd, 0x97, 0x5d, 0x6f, 0xd9, 0x7d, 0x96, 0x46, 0x15, 0xf2, 0x76, 0xc0, 0x37, 0x51,
	0x68, 0xb5, 0x0d, 0xd5, 0x23, 0x05, 0xf3, 0x88, 0x0c, 0xe5, 0xa5, 0xcd, 0x57, 0x1b, 0x0b, 0xa3,
	0x87, 0x99, 0xd6, 0xd4, 0x4b, 0xa8, 0x8c, 0x29, 0x3f, 0x04, 0x79, 0x76, 0xe1, 0x8c, 0xf0, 0x73,
	0xa8, 0x1e, 0xaf, 0x23, 0xe1, 0xc7, 0x67, 0x49, 0xbd, 0x8a, 0x93, 0x4a, 0xeb, 0xac, 0xbe, 0x0f,
	0x35, 0x7c, 0x66, 0xb8, 0x81, 0xbf, 0xa2, 0x2c, 0x8c, 0x93, 0xb0, 0x0a, 0xb9, 0xd4, 0x0e, 0xd4,
	0x4f, 0xdd, 0xc8, 0xab, 0x40, 0x1e, 0xd5, 0x4c, 0x58, 0x21, 0x64, 0x8d, 0xdc, 0xea, 0x27, 0x50,
	0x8c, 0x4e, 0x61, 0xae, 0xa1, 0x5a, 0x8e, 0xbd, 0xa7, 0xb1, 0xec, 0xa1, 0xac, 0x5e, 0x9c, 0xe9,
	0xdf, 0x19, 0x68, 0x3e, 0x4b, 0x84, 0x7c, 0x08, 0x45, 0xec, 0x05, 0x0f, 0x98, 0x78, 0x51, 0xef,
	0x37, 0x0e, 0x22, 0x1a, 0x02, 0x4e, 0xda, 0x98, 0x3d, 0x6d, 0x63, 0x4e, 0x18, 0x0d, 0x28, 0xf9,
	0x81, 0xc3, 0xb7, 0x18, 0x2d, 0x6c, 0x5c, 0x2e, 0xec, 0xab, 0xef, 0x5b, 0x5b, 0xc6, 0x02, 0xba,
	0x16, 0x2d, 0x2b, 0x84, 0x10, 0x67, 0x09, 0x54, 0x14, 0xd0, 0x0b, 0xa8, 0x30, 0xee, 0x7a, 0x96,
	0x8f, 0x4d, 0x72, 0x1d, 0xa5, 0x24, 0xb8, 0xb0, 0xfc, 0xb5, 0xcd, 0xa9, 0x52, 0x16, 0x56, 0x0b,
	0xf2, 0x9e, 0x8d, 0x62, 0x48, 0xa2, 0xd8, 0xea, 0x21, 0xa5, 0x6f, 0x5c, 0x2f, 0x9e, 0x14, 0x10,
	0xad, 0x40, 0xaa, 0x3b, 0x7b, 0xbb, 0x4b, 0xa8, 0x2a, 0xa2, 0xce, 0x2f, 0xa0, 0x7c, 0xb8, 0x8c,
	0xf9, 0x6e, 0x5c, 0x2f, 0x16, 0xe3, 0x03, 0xc8, 0xf3, 0x27, 0x8f, 0x8a, 0x52, 0xea, 0x7d, 0xf9,
	0x94, 0xd5, 0x44, 0x5c, 0xfd, 0x11, 0x24, 0x61, 0x2f, 0x18, 0xf5, 0xe3, 0x30, 0x87, 0x8e, 0x0b,
	0x4d, 0x23, 0x0d, 0x6a, 0x50, 0xa0, 0x7b, 0x8c, 0x1a, 0xab, 0x80, 0x21, 0xf6, 0xf6, 0xcf, 0xf1,
	0xe8, 0xa2, 0x6f, 0x4d, 0x77, 0xf6, 0x93, 0xa8, 0xbe, 0x16, 0xfa, 0x1e, 0xe8, 0x93, 0xa8, 0x5b,
	0xea, 0xfc, 0x91, 0xc5, 0xd9, 0x3b, 0x89, 0x47, 0x24, 0x28, 0x8c, 0x16, 0xd3, 0xe9, 0x8d, 0xfc,
	0x0e, 0x16, 0xd2, 0x18, 0x19, 0xa6, 0xa5, 0x6b, 0xdf, 0x5b, 0x86, 0x36, 0x9e, 0x6a, 0x33, 0x53,
	0xce, 0xe0, 0x3f, 0x78, 0x11, 0x82, 0xc6, 0xcd, 0x34, 0xe5, 0xc8, 0x92, 0xf7, 0xe0, 0x95, 0xa9,
	0x5b, 0xe6, 0xdc, 0x32, 0xf4, 0x61, 0xca, 0x95, 0x23, 0x04, 0xea, 0xba, 0x9e, 0xc2, 0xf2, 0x38,
	0x3a, 0x2f, 0x8d, 0xeb, 0xf9, 0xfc, 0x6b, 0xeb, 0xcc, 0x53, 0x20, 0xef, 0x02, 0x31, 0x0d, 0x6b,
	0x30, 0xfa, 0x2e, 0x85, 0x17, 0xc9, 0x1b, 0x50, 0xa2, 0x17, 0xff, 0xe1, 0x2d, 0xe1, 0xbf, 0x6b,
	0xff, 0x9f, 0x37, 0x84, 0x6e, 0x35, 0x7d, 0x2e, 0x97, 0xc9, 0xa7, 0x70, 0xf9, 0xb6, 0x5b, 0xd6,
	0x68, 0xbe, 0xb8, 0x9a, 0x68, 0x96, 0x61, 0x0e, 0xa6, 0xd7, 0xb2, 0xd4, 0x19, 0xe1, 0xff, 0x4e,
	0x4d, 0x5e, 0x49, 0xe8, 0x63, 0xfd, 0x80, 0x0a, 0xa1, 0xa1, 0x2f, 0x66, 0xb3, 0x6f, 0x67, 0x63,
	0x54, 0xa6, 0x06, 0xd2, 0x70, 0x3e, 0xbd, 0x9e, 0x68, 0xa6, 0x36, 0x42, 0x3d, 0xaa, 0x50, 0x1e,
	0x0e, 0x66, 0x43, 0x6d, 0x82, 0x56, 0xae, 0xff, 0x4f, 0x06, 0x0a, 0x82, 0x86, 0x8c, 0x71, 0x12,
	0xe2, 0xcf, 0x4e, 0xa2, 0xb9, 0x3e, 0x6e, 0x8f, 0x96, 0x9c, 0x06, 0x98, 0xa7, 0x2a, 0xbf, 0xfd,
	0xf9, 0xd7, 0xef, 0x59, 0xa2, 0xd6, 0xc4, 0x92, 0x7b, 0xec, 0x47, 0x3b, 0xf0, 0xab, 0x4c, 0x87,
	0xcc, 0xa1, 0x9c, 0x7c, 0xf2, 0x88, 0xe8, 0x64, 0x43, 0x44, 0x44, 0xa7, 0x3b, 0x40, 0x6d, 0x0b,
	0xa2, 0x16, 0x51, 0x52, 0x44, 0xbd, 0x5f, 0x92, 0x45, 0xf2, 0x2b, 0x99, 0x00, 0x1c, 0xff, 0x37,
	0x69, 0xc6, 0x0c, 0xc7, 0x75, 0xd0, 0x22, 0xe7, 0x10, 0xd2, 0xbe, 0x16, 0xb4, 0x4d, 0xd2, 0x48,
	0x68, 0x59, 0xe4, 0xbc, 0xca, 0xdf, 0xe2, 0x86, 0x5e, 0x16, 0xc5, 0x2a, 0xfe, 0xec, 0xdf, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x1a, 0x6f, 0xc0, 0xf0, 0xe9, 0x05, 0x00, 0x00,
}
