// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	flow "github.com/onflow/flow-go/model/flow"

	mock "github.com/stretchr/testify/mock"

	model "github.com/onflow/flow-go/consensus/hotstuff/model"
)

// SignerVerifier is an autogenerated mock type for the SignerVerifier type
type SignerVerifier struct {
	mock.Mock
}

// CreateProposal provides a mock function with given fields: block
func (_m *SignerVerifier) CreateProposal(block *model.Block) (*model.Proposal, error) {
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
func (_m *SignerVerifier) CreateQC(votes []*model.Vote) (*flow.QuorumCertificate, error) {
	ret := _m.Called(votes)

	var r0 *flow.QuorumCertificate
	if rf, ok := ret.Get(0).(func([]*model.Vote) *flow.QuorumCertificate); ok {
		r0 = rf(votes)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*flow.QuorumCertificate)
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
func (_m *SignerVerifier) CreateVote(block *model.Block) (*model.Vote, error) {
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
func (_m *SignerVerifier) VerifyQC(voterIDs []flow.Identifier, sigData []byte, block *model.Block) (bool, error) {
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
func (_m *SignerVerifier) VerifyVote(voterID flow.Identifier, sigData []byte, block *model.Block) (bool, error) {
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
