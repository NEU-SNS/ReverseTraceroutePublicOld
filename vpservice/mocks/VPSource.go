package mocks

import "github.com/stretchr/testify/mock"

import dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"

type VPSource struct {
	mock.Mock
}

// GetVPs provides a mock function with given fields:
func (_m *VPSource) GetVPs() (*dm.VPReturn, error) {
	ret := _m.Called()

	var r0 *dm.VPReturn
	if rf, ok := ret.Get(0).(func() *dm.VPReturn); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dm.VPReturn)
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

// GetOneVPPerSite provides a mock function with given fields:
func (_m *VPSource) GetOneVPPerSite() (*dm.VPReturn, error) {
	ret := _m.Called()

	var r0 *dm.VPReturn
	if rf, ok := ret.Get(0).(func() *dm.VPReturn); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dm.VPReturn)
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
