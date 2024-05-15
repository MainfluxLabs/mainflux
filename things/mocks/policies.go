package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.RolesRepository = (*rolesRepositoryMock)(nil)

type rolesRepositoryMock struct {
	mu             sync.Mutex
	groupRoles     map[string]things.GroupMembers
	groupRolesByID map[string]things.GroupRoles
}

// NewRolesRepository returns mock of policies repository
func NewRolesRepository() things.RolesRepository {
	return &rolesRepositoryMock{
		groupRoles:     make(map[string]things.GroupMembers),
		groupRolesByID: make(map[string]things.GroupRoles),
	}
}

func (mrm *rolesRepositoryMock) SaveRolesByGroup(ctx context.Context, groupID string, gps ...things.GroupRoles) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, gp := range gps {
		mrm.groupRoles[groupID] = things.GroupMembers{
			MemberID: gp.MemberID,
			Role:     gp.Role,
		}
		mrm.groupRolesByID[gp.MemberID] = gp
	}

	return nil
}

func (mrm *rolesRepositoryMock) RetrieveRole(ctx context.Context, gp things.GroupMembers) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	return mrm.groupRolesByID[gp.MemberID].Role, nil
}

func (mrm *rolesRepositoryMock) RetrieveRolesByGroup(ctx context.Context, groupID string, pm things.PageMetadata) (things.GroupRolesPage, error) {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RetrieveAllRolesByGroup(ctx context.Context) ([]things.GroupMembers, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var gps []things.GroupMembers
	for _, gp := range mrm.groupRoles {
		gps = append(gps, gp)
	}

	return gps, nil
}
func (mrm *rolesRepositoryMock) UpdateRolesByGroup(ctx context.Context, groupID string, gps ...things.GroupRoles) error {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RemoveRolesByGroup(ctx context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
