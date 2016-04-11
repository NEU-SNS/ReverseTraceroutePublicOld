package mocks

import "github.com/stretchr/testify/mock"

import "github.com/NEU-SNS/ReverseTraceroute/datamodel"

type AdjacencySource struct {
	mock.Mock
}

// GetAdjacenciesByIP1 provides a mock function with given fields: _a0
func (_m *AdjacencySource) GetAdjacenciesByIP1(_a0 uint32) ([]datamodel.Adjacency, error) {
	ret := _m.Called(_a0)

	var r0 []datamodel.Adjacency
	if rf, ok := ret.Get(0).(func(uint32) []datamodel.Adjacency); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]datamodel.Adjacency)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAdjacenciesByIP2 provides a mock function with given fields: _a0
func (_m *AdjacencySource) GetAdjacenciesByIP2(_a0 uint32) ([]datamodel.Adjacency, error) {
	ret := _m.Called(_a0)

	var r0 []datamodel.Adjacency
	if rf, ok := ret.Get(0).(func(uint32) []datamodel.Adjacency); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]datamodel.Adjacency)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAdjacencyToDestByAddrAndDest24 provides a mock function with given fields: _a0, _a1
func (_m *AdjacencySource) GetAdjacencyToDestByAddrAndDest24(_a0 uint32, _a1 uint32) ([]datamodel.AdjacencyToDest, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []datamodel.AdjacencyToDest
	if rf, ok := ret.Get(0).(func(uint32, uint32) []datamodel.AdjacencyToDest); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]datamodel.AdjacencyToDest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, uint32) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
