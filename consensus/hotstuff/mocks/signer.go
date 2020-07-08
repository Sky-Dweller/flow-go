// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	flow "github.com/dapperlabs/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"

	model "github.com/dapperlabs/flow-go/consensus/hotstuff/model"
)

// Signer is an autogenerated mock type for the Signer type
type Signer struct {
	mock.Mock
}

// CreateProposal provides a mock function with given fields: block
func (_m *Signer) CreateProposal(block *model.Block) (*model.Proposal, error) {
	ret := _m.Called(block)

	var r0 *model.Proposal
	if rf, ok := ret.Get(0).(func(*model.Block) *model.Proposal); ok {
		r0 = rf(block)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Proposal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.Block) error); ok {
		r1 = rf(block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateQC provides a mock function with given fields: votes
func (_m *Signer) CreateQC(votes []*model.Vote) (*model.QuorumCertificate, error) {
	ret := _m.Called(votes)

	var r0 *model.QuorumCertificate
	if rf, ok := ret.Get(0).(func([]*model.Vote) *model.QuorumCertificate); ok {
		r0 = rf(votes)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.QuorumCertificate)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]*model.Vote) error); ok {
		r1 = rf(votes)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateVote provides a mock function with given fields: block
func (_m *Signer) CreateVote(block *model.Block) (*model.Vote, error) {
	ret := _m.Called(block)

	var r0 *model.Vote
	if rf, ok := ret.Get(0).(func(*model.Block) *model.Vote); ok {
		r0 = rf(block)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.Vote)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.Block) error); ok {
		r1 = rf(block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VerifyQC provides a mock function with given fields: voterIDs, sigData, block
func (_m *Signer) VerifyQC(voterIDs []flow.Identifier, sigData []byte, block *model.Block) (bool, error) {
	ret := _m.Called(voterIDs, sigData, block)

	var r0 bool
	if rf, ok := ret.Get(0).(func([]flow.Identifier, []byte, *model.Block) bool); ok {
		r0 = rf(voterIDs, sigData, block)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]flow.Identifier, []byte, *model.Block) error); ok {
		r1 = rf(voterIDs, sigData, block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// VerifyVote provides a mock function with given fields: voterID, sigData, block
func (_m *Signer) VerifyVote(voterID flow.Identifier, sigData []byte, block *model.Block) (bool, error) {
	ret := _m.Called(voterID, sigData, block)

	var r0 bool
	if rf, ok := ret.Get(0).(func(flow.Identifier, []byte, *model.Block) bool); ok {
		r0 = rf(voterID, sigData, block)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(flow.Identifier, []byte, *model.Block) error); ok {
		r1 = rf(voterID, sigData, block)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
