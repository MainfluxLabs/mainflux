package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
)

var _ auth.PoliciesRepository = (*policiesRepositoryMock)(nil)

type policiesRepositoryMock struct {
	mu                sync.Mutex
	groupPolicies     map[string]auth.GroupPolicy
	groupPoliciesByID map[string]auth.GroupPolicyByID
}

// NewPoliciesRepository returns mock of policies repository
func NewPoliciesRepository() auth.PoliciesRepository {
	return &policiesRepositoryMock{
		groupPolicies:     make(map[string]auth.GroupPolicy),
		groupPoliciesByID: make(map[string]auth.GroupPolicyByID),
	}
}

func (mrm *policiesRepositoryMock) SaveGroupPolicies(ctx context.Context, groupID string, gps ...auth.GroupPolicyByID) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, gp := range gps {
		mrm.groupPolicies[groupID] = auth.GroupPolicy{
			MemberID: gp.MemberID,
			Policy:   gp.Policy,
		}
		mrm.groupPoliciesByID[gp.MemberID] = gp
	}

	return nil
}

func (mrm *policiesRepositoryMock) RetrieveGroupPolicy(ctx context.Context, gp auth.GroupPolicy) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	return mrm.groupPoliciesByID[gp.MemberID].Policy, nil
}

func (mrm *policiesRepositoryMock) RetrieveGroupPolicies(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupPoliciesPage, error) {
	panic("not implemented")
}

func (mrm *policiesRepositoryMock) RetrieveAllGroupPolicies(ctx context.Context) ([]auth.GroupPolicy, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var gps []auth.GroupPolicy
	for _, gp := range mrm.groupPolicies {
		gps = append(gps, gp)
	}

	return gps, nil
}
func (mrm *policiesRepositoryMock) UpdateGroupPolicies(ctx context.Context, groupID string, gps ...auth.GroupPolicyByID) error {
	panic("not implemented")
}

func (mrm *policiesRepositoryMock) RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
