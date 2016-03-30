package mocks

import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
import "github.com/stretchr/testify/mock"

type AtlasServer struct {
	mock.Mock
}

// GetIntersectingPath provides a mock function with given fields: _a0
func (_m *AtlasServer) GetIntersectingPath(_a0 pb.Atlas_GetIntersectingPathServer) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(pb.Atlas_GetIntersectingPathServer) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetPathsWithToken provides a mock function with given fields: _a0
func (_m *AtlasServer) GetPathsWithToken(_a0 pb.Atlas_GetPathsWithTokenServer) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(pb.Atlas_GetPathsWithTokenServer) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
