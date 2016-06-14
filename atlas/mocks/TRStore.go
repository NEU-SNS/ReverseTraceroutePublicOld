package mocks

import "github.com/NEU-SNS/ReverseTraceroute/atlas/types"
import "github.com/stretchr/testify/mock"

import "time"
import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
import "github.com/NEU-SNS/ReverseTraceroute/datamodel"

type TRStore struct {
	mock.Mock
}

// FindIntersectingTraceroute provides a mock function with given fields: _a0
func (_m *TRStore) FindIntersectingTraceroute(_a0 types.IntersectionQuery) (*pb.Path, error) {
	ret := _m.Called(_a0)

	var r0 *pb.Path
	if rf, ok := ret.Get(0).(func(types.IntersectionQuery) *pb.Path); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Path)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(types.IntersectionQuery) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreAtlasTraceroute provides a mock function with given fields: _a0
func (_m *TRStore) StoreAtlasTraceroute(_a0 *datamodel.Traceroute) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.Traceroute) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAtlasSources provides a mock function with given fields: _a0, _a1
func (_m *TRStore) GetAtlasSources(_a0 uint32, _a1 time.Duration) ([]uint32, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []uint32
	if rf, ok := ret.Get(0).(func(uint32, time.Duration) []uint32); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]uint32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, time.Duration) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
