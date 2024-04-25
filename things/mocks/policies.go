package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.PoliciesRepository = (*policiesRepositoryMock)(nil)

type policiesRepositoryMock struct {
	mu                sync.Mutex
	groupPolicies     map[string]things.GroupPolicy
	groupPoliciesByID map[string]things.GroupPolicyByID
}

// NewPoliciesRepository returns mock of policies repository
func NewPoliciesRepository() things.PoliciesRepository {
	return &policiesRepositoryMock{
		groupPolicies:     make(map[string]things.GroupPolicy),
		groupPoliciesByID: make(map[string]things.GroupPolicyByID),
	}
}

func (mrm *policiesRepositoryMock) SaveGroupPolicies(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, gp := range gps {
		mrm.groupPolicies[groupID] = things.GroupPolicy{
			MemberID: gp.MemberID,
			Policy:   gp.Policy,
		}
		mrm.groupPoliciesByID[gp.MemberID] = gp
	}

	return nil
}

func (mrm *policiesRepositoryMock) RetrieveGroupPolicy(ctx context.Context, gp things.GroupPolicy) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	return mrm.groupPoliciesByID[gp.MemberID].Policy, nil
}

func (mrm *policiesRepositoryMock) RetrieveGroupPolicies(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupPoliciesPage, error) {
	panic("not implemented")
}

func (mrm *policiesRepositoryMock) RetrieveAllGroupPolicies(ctx context.Context) ([]things.GroupPolicy, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var gps []things.GroupPolicy
	for _, gp := range mrm.groupPolicies {
		gps = append(gps, gp)
	}

	return gps, nil
}
func (mrm *policiesRepositoryMock) UpdateGroupPolicies(ctx context.Context, groupID string, gps ...things.GroupPolicyByID) error {
	panic("not implemented")
}

func (mrm *policiesRepositoryMock) RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
