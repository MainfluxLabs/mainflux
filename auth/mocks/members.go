package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.MembersRepository = (*membersRepositoryMock)(nil)

type membersRepositoryMock struct {
	mu             sync.Mutex
	members        map[string][]auth.OrgMember
	membersByOrgID map[string][]auth.OrgMember
}

// NewMembersRepository returns mock of org repository
func NewMembersRepository() auth.MembersRepository {
	return &membersRepositoryMock{
		members:        make(map[string][]auth.OrgMember),
		membersByOrgID: make(map[string][]auth.OrgMember),
	}
}

func (mrm *membersRepositoryMock) Save(ctx context.Context, oms ...auth.OrgMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, om := range oms {
		if om.OrgID == "" {
			return errors.ErrNotFound
		}

		m := auth.OrgMember{
			MemberID: om.MemberID,
			Role:     om.Role,
			OrgID:    om.OrgID,
		}

		mrm.members[om.MemberID] = append(mrm.members[om.MemberID], m)
		mrm.membersByOrgID[om.OrgID] = append(mrm.membersByOrgID[om.OrgID], m)
	}

	return nil
}

func (mrm *membersRepositoryMock) Remove(ctx context.Context, orgID string, memberIDs ...string) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, memberID := range memberIDs {
		if _, ok := mrm.members[memberID]; !ok {
			return errors.ErrNotFound
		}

		if _, ok := mrm.membersByOrgID[orgID]; !ok {
			return errors.ErrNotFound
		}

		delete(mrm.members, memberID)
	}

	return nil
}

func (mrm *membersRepositoryMock) Update(ctx context.Context, oms ...auth.OrgMember) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, om := range oms {
		if _, ok := mrm.members[om.MemberID]; !ok {
			return errors.ErrNotFound
		}
		mrm.members[om.MemberID] = []auth.OrgMember{
			{
				MemberID: om.MemberID,
				Role:     om.Role,
			},
		}
	}

	return nil
}

func (mrm *membersRepositoryMock) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	if members, ok := mrm.members[memberID]; ok {
		for _, m := range members {
			if m.OrgID == orgID {
				return m.Role, nil
			}
		}
	}

	return "", errors.ErrNotFound
}

func (mrm *membersRepositoryMock) RetrieveByOrgID(ctx context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembersPage, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	oms := []auth.OrgMember{}
	i := uint64(0)
	for _, m := range mrm.membersByOrgID[orgID] {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			oms = append(oms, m)
		}
		i++
	}

	return auth.OrgMembersPage{
		OrgMembers: oms,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(mrm.membersByOrgID[orgID])),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (mrm *membersRepositoryMock) RetrieveAll(ctx context.Context) ([]auth.OrgMember, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	oms := []auth.OrgMember{}
	for _, members := range mrm.members {
		oms = append(oms, members...)
	}

	return oms, nil
}
