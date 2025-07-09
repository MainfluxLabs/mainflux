package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
)

var _ auth.OrgMembershipsRepository = (*orgMembershipsRepositoryMock)(nil)

type orgMembershipsRepositoryMock struct {
	mu                 sync.Mutex
	memberships        map[string][]auth.OrgMembership
	membershipsByOrgID map[string][]auth.OrgMembership
}

// NewOrgMembershipsRepository returns mock of org memberships repository
func NewOrgMembershipsRepository() auth.OrgMembershipsRepository {
	return &orgMembershipsRepositoryMock{
		memberships:        make(map[string][]auth.OrgMembership),
		membershipsByOrgID: make(map[string][]auth.OrgMembership),
	}
}

func (mrm *orgMembershipsRepositoryMock) Save(_ context.Context, oms ...auth.OrgMembership) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, om := range oms {
		if om.OrgID == "" {
			return errors.ErrNotFound
		}

		m := auth.OrgMembership{
			MemberID: om.MemberID,
			Role:     om.Role,
			OrgID:    om.OrgID,
		}

		mrm.memberships[om.MemberID] = append(mrm.memberships[om.MemberID], m)
		mrm.membershipsByOrgID[om.OrgID] = append(mrm.membershipsByOrgID[om.OrgID], m)
	}

	return nil
}

func (mrm *orgMembershipsRepositoryMock) Remove(_ context.Context, orgID string, memberIDs ...string) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, memberID := range memberIDs {
		if _, ok := mrm.memberships[memberID]; !ok {
			return errors.ErrNotFound
		}

		if _, ok := mrm.membershipsByOrgID[orgID]; !ok {
			return errors.ErrNotFound
		}

		delete(mrm.memberships, memberID)
	}

	return nil
}

func (mrm *orgMembershipsRepositoryMock) Update(_ context.Context, oms ...auth.OrgMembership) error {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	for _, om := range oms {
		if _, ok := mrm.memberships[om.MemberID]; !ok {
			return errors.ErrNotFound
		}
		mrm.memberships[om.MemberID] = []auth.OrgMembership{
			{
				MemberID: om.MemberID,
				Role:     om.Role,
			},
		}
	}

	return nil
}

func (mrm *orgMembershipsRepositoryMock) RetrieveRole(_ context.Context, memberID, orgID string) (string, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	if memberships, ok := mrm.memberships[memberID]; ok {
		for _, m := range memberships {
			if m.OrgID == orgID {
				return m.Role, nil
			}
		}
	}

	return "", errors.ErrNotFound
}

func (mrm *orgMembershipsRepositoryMock) RetrieveByOrgID(_ context.Context, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	memberships := mrm.membershipsByOrgID[orgID]

	sortedMemberships := mocks.SortItems(pm.Order, pm.Dir, memberships, func(i int) (string, string) {
		return memberships[i].Email, memberships[i].MemberID
	})

	var oms []auth.OrgMembership
	i := uint64(0)
	for _, m := range sortedMemberships {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			oms = append(oms, m)
		}
		i++
	}

	return auth.OrgMembershipsPage{
		OrgMemberships: oms,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(mrm.membershipsByOrgID[orgID])),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (mrm *orgMembershipsRepositoryMock) BackupAll(_ context.Context) ([]auth.OrgMembership, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var oms []auth.OrgMembership
	for _, m := range mrm.memberships {
		oms = append(oms, m...)
	}

	return oms, nil
}
