package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.RolesRepository = (*rolesRepositoryMock)(nil)

type rolesRepositoryMock struct {
	mu             sync.Mutex
	groupRoles     map[string]things.GroupMember
	groupRolesByID map[string]things.GroupRoles
}

// NewRolesRepository returns mock of roles repository
func NewRolesRepository() things.RolesRepository {
	return &rolesRepositoryMock{
		groupRoles:     make(map[string]things.GroupMember),
		groupRolesByID: make(map[string]things.GroupRoles),
	}
}

func (mrm *rolesRepositoryMock) SaveRolesByGroup(_ context.Context, groupID string, gps ...things.GroupRoles) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, gp := range gps {
		mrm.groupRoles[groupID] = things.GroupMember{
			MemberID: gp.MemberID,
			Role:     gp.Role,
		}
		mrm.groupRolesByID[gp.MemberID] = gp
	}

	return nil
}

func (mrm *rolesRepositoryMock) RetrieveRole(_ context.Context, gp things.GroupMember) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	return mrm.groupRolesByID[gp.MemberID].Role, nil
}

func (mrm *rolesRepositoryMock) RetrieveRolesByGroup(_ context.Context, groupID string, pm things.PageMetadata) (things.GroupMembersPage, error) {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var grIDs []string
	for k, gr := range mrm.groupRoles {
		if gr.MemberID == memberID {
			grIDs = append(grIDs, k)
		}
	}

	return grIDs, nil
}

func (mrm *rolesRepositoryMock) RetrieveAllRolesByGroup(_ context.Context) ([]things.GroupMember, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var gps []things.GroupMember
	for _, gp := range mrm.groupRoles {
		gps = append(gps, gp)
	}

	return gps, nil
}
func (mrm *rolesRepositoryMock) UpdateRolesByGroup(_ context.Context, groupID string, gps ...things.GroupRoles) error {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RemoveRolesByGroup(_ context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
