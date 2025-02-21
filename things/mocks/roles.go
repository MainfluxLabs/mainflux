package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.RolesRepository = (*rolesRepositoryMock)(nil)

type rolesRepositoryMock struct {
	mu         sync.Mutex
	groupRoles map[string][]things.GroupMember
}

// NewRolesRepository returns mock of roles repository
func NewRolesRepository() things.RolesRepository {
	return &rolesRepositoryMock{
		groupRoles: make(map[string][]things.GroupMember),
	}
}

func (mrm *rolesRepositoryMock) SaveRolesByGroup(_ context.Context, gms ...things.GroupMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, g := range gms {
		mrm.groupRoles[g.GroupID] = append(mrm.groupRoles[g.GroupID], g)
	}

	return nil
}

func (mrm *rolesRepositoryMock) RetrieveRole(_ context.Context, gm things.GroupMember) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, mbr := range mrm.groupRoles[gm.GroupID] {
		if mbr.MemberID == gm.MemberID {
			return mbr.Role, nil
		}
	}

	return "", errors.ErrNotFound
}

func (mrm *rolesRepositoryMock) RetrieveRolesByGroup(_ context.Context, groupID string, pm things.PageMetadata) (things.GroupMembersPage, error) {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var grIDs []string
	for grID, mbrs := range mrm.groupRoles {
		for _, gr := range mbrs {
			if gr.MemberID == memberID {
				grIDs = append(grIDs, grID)
				break
			}
		}
	}

	return grIDs, nil
}

func (mrm *rolesRepositoryMock) RetrieveAllRolesByGroup(_ context.Context) ([]things.GroupMember, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var mbrs []things.GroupMember
	for _, mb := range mrm.groupRoles {
		mbrs = append(mbrs, mb...)
	}

	return mbrs, nil
}

func (mrm *rolesRepositoryMock) UpdateRolesByGroup(_ context.Context, gms ...things.GroupMember) error {
	panic("not implemented")
}

func (mrm *rolesRepositoryMock) RemoveRolesByGroup(_ context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
