package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.GroupMembersRepository = (*groupMembersRepositoryMock)(nil)

type groupMembersRepositoryMock struct {
	mu         sync.Mutex
	groupRoles map[string][]things.GroupMember
}

// NewGroupMembersRepository returns mock of roles repository
func NewGroupMembersRepository() things.GroupMembersRepository {
	return &groupMembersRepositoryMock{
		groupRoles: make(map[string][]things.GroupMember),
	}
}

func (mrm *groupMembersRepositoryMock) Save(_ context.Context, gms ...things.GroupMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, g := range gms {
		mrm.groupRoles[g.GroupID] = append(mrm.groupRoles[g.GroupID], g)
	}

	return nil
}

func (mrm *groupMembersRepositoryMock) RetrieveRole(_ context.Context, gm things.GroupMember) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, mbr := range mrm.groupRoles[gm.GroupID] {
		if mbr.MemberID == gm.MemberID {
			return mbr.Role, nil
		}
	}

	return "", errors.ErrNotFound
}

func (mrm *groupMembersRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembersPage, error) {
	panic("not implemented")
}

func (mrm *groupMembersRepositoryMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
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

func (mrm *groupMembersRepositoryMock) RetrieveAll(_ context.Context) ([]things.GroupMember, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var mbrs []things.GroupMember
	for _, mb := range mrm.groupRoles {
		mbrs = append(mbrs, mb...)
	}

	return mbrs, nil
}

func (mrm *groupMembersRepositoryMock) UpdateRolesByGroup(_ context.Context, gms ...things.GroupMember) error {
	panic("not implemented")
}

func (mrm *groupMembersRepositoryMock) RemoveRolesByGroup(_ context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
