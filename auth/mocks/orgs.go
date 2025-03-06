package mocks

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var _ auth.OrgRepository = (*orgRepositoryMock)(nil)

type orgRepositoryMock struct {
	mu      sync.Mutex
	orgs    map[string]auth.Org
	members auth.MembersRepository
}

// NewOrgRepository returns mock of org repository
func NewOrgRepository(mr auth.MembersRepository) auth.OrgRepository {
	return &orgRepositoryMock{
		orgs:    make(map[string]auth.Org),
		members: mr,
	}
}

func (orm *orgRepositoryMock) Save(ctx context.Context, orgs ...auth.Org) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, org := range orgs {
		if _, ok := orm.orgs[org.ID]; ok {
			return errors.ErrConflict
		}

		orm.orgs[org.ID] = org
	}

	return nil
}

func (orm *orgRepositoryMock) Update(ctx context.Context, org auth.Org) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	if _, ok := orm.orgs[org.ID]; !ok {
		return errors.ErrNotFound
	}

	orm.orgs[org.ID] = org

	return nil
}

func (orm *orgRepositoryMock) Remove(ctx context.Context, owner, id string) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	if _, ok := orm.orgs[id]; !ok && orm.orgs[id].OwnerID != owner {
		return errors.ErrNotFound
	}
	delete(orm.orgs, id)

	return nil
}

func (orm *orgRepositoryMock) RetrieveByID(ctx context.Context, id string) (auth.Org, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	org, ok := orm.orgs[id]
	if !ok {
		return auth.Org{}, errors.ErrNotFound
	}

	return org, nil
}

func (orm *orgRepositoryMock) RetrieveByOwner(ctx context.Context, ownerID string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()
	keys := sortOrgsByID(orm.orgs)

	i := uint64(0)
	orgs := make([]auth.Org, 0)
	for _, k := range keys {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if orm.orgs[k].OwnerID == ownerID && strings.Contains(orm.orgs[k].Name, pm.Name) {
				orgs = append(orgs, orm.orgs[k])
			}
		}
		i++
	}

	return auth.OrgsPage{
		Orgs: orgs,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveByMemberID(ctx context.Context, memberID string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	members, _ := orm.members.RetrieveAll(ctx)
	orgs := []auth.Org{}

	first := uint64(pm.Offset) + 1
	last := first + pm.Limit
	for _, m := range members {
		for _, o := range orm.orgs {
			if m.MemberID == memberID && m.OrgID == o.ID {
				id := uuid.ParseID(o.ID)
				if id >= first && id < last {
					orgs = append(orgs, orm.orgs[m.OrgID])
				}
			}
		}
	}

	if pm.Name != "" {
		var filteredItems []auth.Org
		for _, o := range orgs {
			if strings.Contains(o.Name, pm.Name) {
				filteredItems = append(filteredItems, o)
			}
		}
		orgs = filteredItems
	}

	orgs = mocks.SortItems(pm.Order, pm.Dir, orgs, func(i int) (string, string) {
		return orgs[i].Name, orgs[i].ID
	})

	return auth.OrgsPage{
		Orgs: orgs,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveAll(ctx context.Context) ([]auth.Org, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	var orgs []auth.Org
	for _, org := range orm.orgs {
		orgs = append(orgs, org)
	}

	return orgs, nil
}

func (orm *orgRepositoryMock) RetrieveByAdmin(ctx context.Context, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	keys := sortOrgsByID(orm.orgs)

	orgs := make([]auth.Org, 0)
	for _, k := range keys[pm.Offset : pm.Offset+pm.Limit] {
		// filter by name
		if strings.Contains(orm.orgs[k].Name, pm.Name) {
			orgs = append(orgs, orm.orgs[k])
		}
	}

	return auth.OrgsPage{
		Orgs: orgs,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func sortOrgsByID(orgs map[string]auth.Org) []string {
	var keys []string
	for k := range orgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
