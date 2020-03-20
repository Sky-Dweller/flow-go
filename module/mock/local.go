// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import crypto "github.com/dapperlabs/flow-go/crypto"
import flow "github.com/dapperlabs/flow-go/model/flow"
import mock "github.com/stretchr/testify/mock"

// Local is an autogenerated mock type for the Local type
type Local struct {
	mock.Mock
}

// Address provides a mock function with given fields:
func (_m *Local) Address() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NodeID provides a mock function with given fields:
func (_m *Local) NodeID() flow.Identifier {
	ret := _m.Called()

	var r0 flow.Identifier
	if rf, ok := ret.Get(0).(func() flow.Identifier); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(flow.Identifier)
		}
	}

	return r0
}

// NotMeFilter provides a mock function with given fields:
func (_m *Local) NotMeFilter() flow.IdentityFilter {
	ret := _m.Called()

	var r0 flow.IdentityFilter
	if rf, ok := ret.Get(0).(func() flow.IdentityFilter); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(flow.IdentityFilter)
		}
	}

	return r0
}

// Sign provides a mock function with given fields: _a0, _a1
func (_m *Local) Sign(_a0 []byte, _a1 crypto.Hasher) (crypto.Signature, error) {
	ret := _m.Called(_a0, _a1)

	var r0 crypto.Signature
	if rf, ok := ret.Get(0).(func([]byte, crypto.Hasher) crypto.Signature); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(crypto.Signature)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte, crypto.Hasher) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
