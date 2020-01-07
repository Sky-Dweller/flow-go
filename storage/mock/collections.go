// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import flow "github.com/dapperlabs/flow-go/model/flow"
import mock "github.com/stretchr/testify/mock"

// Collections is an autogenerated mock type for the Collections type
type Collections struct {
	mock.Mock
}

// ByFingerprint provides a mock function with given fields: hash
func (_m *Collections) ByFingerprint(hash flow.Fingerprint) (*flow.Collection, error) {
	ret := _m.Called(hash)

	var r0 *flow.Collection
	if rf, ok := ret.Get(0).(func(flow.Fingerprint) *flow.Collection); ok {
		r0 = rf(hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.Collection)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Fingerprint) error); ok {
		r1 = rf(hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Remove provides a mock function with given fields: hash
func (_m *Collections) Remove(hash flow.Fingerprint) error {
	ret := _m.Called(hash)

	var r0 error
	if rf, ok := ret.Get(0).(func(flow.Fingerprint) error); ok {
		r0 = rf(hash)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Save provides a mock function with given fields: collection
func (_m *Collections) Save(collection *flow.Collection) error {
	ret := _m.Called(collection)

	var r0 error
	if rf, ok := ret.Get(0).(func(*flow.Collection) error); ok {
		r0 = rf(collection)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
