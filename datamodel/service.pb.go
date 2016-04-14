// Code generated by protoc-gen-go.
// source: github.com/NEU-SNS/ReverseTraceroute/datamodel/service.proto
// DO NOT EDIT!

package datamodel

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type ServiceT int32

const (
	ServiceT_PLANET_LAB ServiceT = 0
)

var ServiceT_name = map[int32]string{
	0: "PLANET_LAB",
}
var ServiceT_value = map[string]int32{
	"PLANET_LAB": 0,
}

func (x ServiceT) String() string {
	return proto.EnumName(ServiceT_name, int32(x))
}
func (ServiceT) EnumDescriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

func init() {
	proto.RegisterEnum("datamodel.ServiceT", ServiceT_name, ServiceT_value)
}

var fileDescriptor3 = []byte{
	// 124 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xb2, 0x49, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xf7, 0x73, 0x0d, 0xd5, 0x0d, 0xf6, 0x0b, 0xd6, 0x0f,
	0x4a, 0x2d, 0x4b, 0x2d, 0x2a, 0x4e, 0x0d, 0x29, 0x4a, 0x4c, 0x4e, 0x2d, 0xca, 0x2f, 0x2d, 0x49,
	0xd5, 0x4f, 0x49, 0x2c, 0x49, 0xcc, 0xcd, 0x4f, 0x49, 0xcd, 0xd1, 0x2f, 0x4e, 0x2d, 0x2a, 0xcb,
	0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x84, 0x4b, 0x68, 0x49, 0x71, 0x71,
	0x04, 0x43, 0xe4, 0x42, 0x84, 0xf8, 0xb8, 0xb8, 0x02, 0x7c, 0x1c, 0xfd, 0x5c, 0x43, 0xe2, 0x7d,
	0x1c, 0x9d, 0x04, 0x18, 0x9c, 0xb8, 0xa3, 0x10, 0x0a, 0x93, 0xd8, 0xc0, 0x5a, 0x8d, 0x01, 0x01,
	0x00, 0x00, 0xff, 0xff, 0x7b, 0xa7, 0x23, 0x24, 0x7a, 0x00, 0x00, 0x00,
}
