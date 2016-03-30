package mocks

import "github.com/stretchr/testify/mock"

import dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
import "golang.org/x/net/context"

type AtlasService struct {
	mock.Mock
}

// GetIntersectingPath provides a mock function with given fields: _a0, _a1
func (_m *AtlasService) GetIntersectingPath(_a0 context.Context, _a1 *dm.IntersectionRequest) ([]*dm.IntersectionResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*dm.IntersectionResponse
	if rf, ok := ret.Get(0).(func(context.Context, *dm.IntersectionRequest) []*dm.IntersectionResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.IntersectionResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dm.IntersectionRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPathsWithToken provides a mock function with given fields: _a0, _a1
func (_m *AtlasService) GetPathsWithToken(_a0 context.Context, _a1 *dm.TokenRequest) ([]*dm.TokenResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []*dm.TokenResponse
	if rf, ok := ret.Get(0).(func(context.Context, *dm.TokenRequest) []*dm.TokenResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*dm.TokenResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *dm.TokenRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
