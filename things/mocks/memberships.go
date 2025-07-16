package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.GroupMembershipsRepository = (*groupMembershipsRepositoryMock)(nil)

type groupMembershipsRepositoryMock struct {
	mu               sync.Mutex
	groupMemberships map[string][]things.GroupMembership
}

// NewGroupMembershipsRepository returns mock of membership repository
func NewGroupMembershipsRepository() things.GroupMembershipsRepository {
	return &groupMembershipsRepositoryMock{
		groupMemberships: make(map[string][]things.GroupMembership),
	}
}

func (gmr *groupMembershipsRepositoryMock) Save(_ context.Context, gms ...things.GroupMembership) error {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	for _, g := range gms {
		gmr.groupMemberships[g.GroupID] = append(gmr.groupMemberships[g.GroupID], g)
	}

	return nil
}

func (gmr *groupMembershipsRepositoryMock) RetrieveRole(_ context.Context, gm things.GroupMembership) (string, error) {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	for _, mbr := range gmr.groupMemberships[gm.GroupID] {
		if mbr.MemberID == gm.MemberID {
			return mbr.Role, nil
		}
	}

	return "", errors.ErrNotFound
}

func (gmr *groupMembershipsRepositoryMock) RetrieveByGroup(_ context.Context, groupID string, pm apiutil.PageMetadata) (things.GroupMembershipsPage, error) {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	memberships := gmr.groupMemberships[groupID]

	sortedMemberships := mocks.SortItems(pm.Order, pm.Dir, memberships, func(i int) (string, string) {
		return memberships[i].Email, memberships[i].MemberID
	})

	var gms []things.GroupMembership
	i := uint64(0)
	for _, m := range sortedMemberships {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			gms = append(gms, m)
		}
		i++
	}

	return things.GroupMembershipsPage{
		GroupMemberships: gms,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(gmr.groupMemberships[groupID])),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil

}

func (gmr *groupMembershipsRepositoryMock) RetrieveGroupIDsByMember(_ context.Context, memberID string) ([]string, error) {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	var grIDs []string
	for grID, gms := range gmr.groupMemberships {
		for _, gr := range gms {
			if gr.MemberID == memberID {
				grIDs = append(grIDs, grID)
				break
			}
		}
	}

	return grIDs, nil
}

func (gmr *groupMembershipsRepositoryMock) BackupAll(_ context.Context) ([]things.GroupMembership, error) {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	var gms []things.GroupMembership
	for _, mb := range gmr.groupMemberships {
		gms = append(gms, mb...)
	}

	return gms, nil
}

func (mrm *groupMembershipsRepositoryMock) BackupByGroup(_ context.Context, groupID string) ([]things.GroupMembership, error) {
	mrm.mu.Lock()
	defer mrm.mu.Unlock()

	var mbrs []things.GroupMembership
	for _, groupMbs := range mrm.groupMemberships {
		for _, mb := range groupMbs {
			if mb.GroupID == groupID {
				mbrs = append(mbrs, mb)
			}
		}
	}

	return mbrs, nil
}

func (gmr *groupMembershipsRepositoryMock) Update(_ context.Context, gms ...things.GroupMembership) error {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	for _, gm := range gms {
		if _, ok := gmr.groupMemberships[gm.GroupID]; !ok {
			return errors.ErrNotFound
		}
		gmr.groupMemberships[gm.GroupID] = []things.GroupMembership{
			{
				MemberID: gm.MemberID,
				Role:     gm.Role,
			},
		}
	}

	return nil
}

func (gmr *groupMembershipsRepositoryMock) Remove(_ context.Context, groupID string, memberIDs ...string) error {
	gmr.mu.Lock()
	defer gmr.mu.Unlock()

	memberships, ok := gmr.groupMemberships[groupID]
	if !ok {
		return errors.ErrNotFound
	}

	for _, memberID := range memberIDs {
		found := false
		for i, membership := range memberships {
			if membership.MemberID == memberID {
				memberships = append(memberships[:i], memberships[i+1:]...)
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
