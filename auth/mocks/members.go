package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
)

var _ auth.MembersRepository = (*membersRepositoryMock)(nil)

type membersRepositoryMock struct {
	mu           sync.Mutex
	groupMembers map[string]auth.GroupInvitationByID
}

// NewMembersRepository returns mock of org repository
func NewMembersRepository() auth.MembersRepository {
	return &membersRepositoryMock{
		groupMembers: make(map[string]auth.GroupInvitationByID),
	}
}

func (mrm *membersRepositoryMock) SaveGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, g := range giByIDs {
		mrm.groupMembers[g.MemberID] = g
	}

	return nil
}

func (mrm *membersRepositoryMock) RetrieveGroupMemberPolicy(ctx context.Context, gp auth.GroupsPolicy) (string, error) {
	panic("not implemented")
}

func (mrm *membersRepositoryMock) RetrieveGroupMembers(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupMembersPage, error) {
	panic("not implemented")
}

func (mrm *membersRepositoryMock) UpdateGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	panic("not implemented")
}

func (mrm *membersRepositoryMock) RemoveGroupMembers(ctx context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}
