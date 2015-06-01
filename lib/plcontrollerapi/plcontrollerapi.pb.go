// Code generated by protoc-gen-go.
// source: github.com/NEU-SNS/ReverseTraceroute/lib/plcontrollerapi/plcontrollerapi.proto
// DO NOT EDIT!

/*
Package plcontrollerapi is a generated protocol buffer package.

It is generated from these files:
	github.com/NEU-SNS/ReverseTraceroute/lib/plcontrollerapi/plcontrollerapi.proto

It has these top-level messages:
*/
package plcontrollerapi

import proto "github.com/golang/protobuf/proto"
import datamodel2 "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
import datamodel4 "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
import datamodel5 "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
import datamodel6 "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"
import datamodel7 "github.com/NEU-SNS/ReverseTraceroute/lib/datamodel"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

func init() {
}

// Client API for PLController service

type PLControllerClient interface {
	Ping(ctx context.Context, in *datamodel4.PingArg, opts ...grpc.CallOption) (*datamodel4.Ping, error)
	Traceroute(ctx context.Context, in *datamodel5.TracerouteArg, opts ...grpc.CallOption) (*datamodel5.Traceroute, error)
	Stats(ctx context.Context, in *datamodel2.StatsArg, opts ...grpc.CallOption) (*datamodel2.Stats, error)
	Register(ctx context.Context, in *datamodel6.VantagePoint, opts ...grpc.CallOption) (*datamodel7.RegisterResponse, error)
}

type pLControllerClient struct {
	cc *grpc.ClientConn
}

func NewPLControllerClient(cc *grpc.ClientConn) PLControllerClient {
	return &pLControllerClient{cc}
}

func (c *pLControllerClient) Ping(ctx context.Context, in *datamodel4.PingArg, opts ...grpc.CallOption) (*datamodel4.Ping, error) {
	out := new(datamodel4.Ping)
	err := grpc.Invoke(ctx, "/.PLController/Ping", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pLControllerClient) Traceroute(ctx context.Context, in *datamodel5.TracerouteArg, opts ...grpc.CallOption) (*datamodel5.Traceroute, error) {
	out := new(datamodel5.Traceroute)
	err := grpc.Invoke(ctx, "/.PLController/Traceroute", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pLControllerClient) Stats(ctx context.Context, in *datamodel2.StatsArg, opts ...grpc.CallOption) (*datamodel2.Stats, error) {
	out := new(datamodel2.Stats)
	err := grpc.Invoke(ctx, "/.PLController/Stats", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pLControllerClient) Register(ctx context.Context, in *datamodel6.VantagePoint, opts ...grpc.CallOption) (*datamodel7.RegisterResponse, error) {
	out := new(datamodel7.RegisterResponse)
	err := grpc.Invoke(ctx, "/.PLController/Register", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for PLController service

type PLControllerServer interface {
	Ping(context.Context, *datamodel4.PingArg) (*datamodel4.Ping, error)
	Traceroute(context.Context, *datamodel5.TracerouteArg) (*datamodel5.Traceroute, error)
	Stats(context.Context, *datamodel2.StatsArg) (*datamodel2.Stats, error)
	Register(context.Context, *datamodel6.VantagePoint) (*datamodel7.RegisterResponse, error)
}

func RegisterPLControllerServer(s *grpc.Server, srv PLControllerServer) {
	s.RegisterService(&_PLController_serviceDesc, srv)
}

func _PLController_Ping_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(datamodel4.PingArg)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(PLControllerServer).Ping(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _PLController_Traceroute_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(datamodel5.TracerouteArg)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(PLControllerServer).Traceroute(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _PLController_Stats_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(datamodel2.StatsArg)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(PLControllerServer).Stats(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _PLController_Register_Handler(srv interface{}, ctx context.Context, codec grpc.Codec, buf []byte) (interface{}, error) {
	in := new(datamodel6.VantagePoint)
	if err := codec.Unmarshal(buf, in); err != nil {
		return nil, err
	}
	out, err := srv.(PLControllerServer).Register(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _PLController_serviceDesc = grpc.ServiceDesc{
	ServiceName: ".PLController",
	HandlerType: (*PLControllerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _PLController_Ping_Handler,
		},
		{
			MethodName: "Traceroute",
			Handler:    _PLController_Traceroute_Handler,
		},
		{
			MethodName: "Stats",
			Handler:    _PLController_Stats_Handler,
		},
		{
			MethodName: "Register",
			Handler:    _PLController_Register_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}
