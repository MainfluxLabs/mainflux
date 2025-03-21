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
	mu           sync.Mutex
	groupMembers map[string][]things.GroupMember
}

// NewGroupMembersRepository returns mock of members repository
func NewGroupMembersRepository() things.GroupMembersRepository {
	return &groupMembersRepositoryMock{
		groupMembers: make(map[string][]things.GroupMember),
	}
}

func (mrm *groupMembersRepositoryMock) Save(_ context.Context, gms ...things.GroupMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, g := range gms {
		mrm.groupMembers[g.GroupID] = append(mrm.groupMembers[g.GroupID], g)
	}

	return nil
}

func (mrm *groupMembersRepositoryMock) RetrieveRole(_ context.Context, gm things.GroupMember) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, mbr := range mrm.groupMembers[gm.GroupID] {
		if mbr.MemberID == gm.MemberID {
			return mbr.Role, nil
		}
	}

	return "", errors.ErrNotFound
}

func (mrm *groupMembersRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembersPage, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	gms := []things.GroupMember{}
	i := uint64(0)
	for _, m := range mrm.groupMembers[groupID] {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			gms = append(gms, m)
		}
		i++
	}

	return things.GroupMembersPage{
		GroupMembers: gms,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(mrm.groupMembers[groupID])),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil

}

func (mrm *groupMembersRepositoryMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var grIDs []string
	for grID, mbrs := range mrm.groupMembers {
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
	for _, mb := range mrm.groupMembers {
		mbrs = append(mbrs, mb...)
	}

	return mbrs, nil
}

func (mrm *groupMembersRepositoryMock) Update(_ context.Context, gms ...things.GroupMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, gm := range gms {
		if _, ok := mrm.groupMembers[gm.GroupID]; !ok {
			return errors.ErrNotFound
		}
		mrm.groupMembers[gm.GroupID] = []things.GroupMember{
			{
				MemberID: gm.MemberID,
				Role:     gm.Role,
			},
		}
	}

	return nil
}

func (mrm *groupMembersRepositoryMock) Remove(_ context.Context, groupID string, memberIDs ...string) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	members, ok := mrm.groupMembers[groupID]
	if !ok {
		return errors.ErrNotFound
	}

	for _, memberID := range memberIDs {
		found := false
		for i, member := range members {
			if member.MemberID == memberID {
				members = append(members[:i], members[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			return errors.ErrNotFound
		}
	}

	return nil
}
