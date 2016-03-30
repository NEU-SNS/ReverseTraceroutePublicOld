package mocks

import "github.com/stretchr/testify/mock"

import datamodel "github.com/NEU-SNS/ReverseTraceroute/datamodel"

type Atlas_GetPathsWithTokenServer struct {
	mock.Mock
}

// Send provides a mock function with given fields: _a0
func (_m *Atlas_GetPathsWithTokenServer) Send(_a0 *datamodel.TokenResponse) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*datamodel.TokenResponse) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Recv provides a mock function with given fields:
func (_m *Atlas_GetPathsWithTokenServer) Recv() (*datamodel.TokenRequest, error) {
	ret := _m.Called()

	var r0 *datamodel.TokenRequest
	if rf, ok := ret.Get(0).(func() *datamodel.TokenRequest); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.TokenRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
