package mocks

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.OrgRepository = (*orgRepositoryMock)(nil)

type orgRepositoryMock struct {
	mu         sync.Mutex
	orgs       map[string]auth.Org
	orgMembers map[string]auth.OrgMember
	orgGroups  map[string]auth.OrgGroup
}

// NewOrgRepository returns mock of org repository
func NewOrgRepository() auth.OrgRepository {
	return &orgRepositoryMock{
		orgs:       make(map[string]auth.Org),
		orgMembers: make(map[string]auth.OrgMember),
		orgGroups:  make(map[string]auth.OrgGroup),
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

func (orm *orgRepositoryMock) Delete(ctx context.Context, owner, id string) error {
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

func (orm *orgRepositoryMock) RetrieveByOwner(ctx context.Context, ownerID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
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
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveOrgsByMember(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	i := uint64(0)
	orgs := make([]auth.Org, 0)
	for _, org := range orm.orgs {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if _, ok := orm.orgMembers[memberID]; ok {
				if strings.Contains(org.Name, pm.Name) {
					orgs = append(orgs, org)
				}
			}
		}
		i++
	}

	return auth.OrgsPage{
		Orgs: orgs,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) AssignMembers(ctx context.Context, oms ...auth.OrgMember) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, om := range oms {
		if _, ok := orm.orgs[om.OrgID]; !ok {
			return errors.ErrNotFound
		}
		orm.orgMembers[om.MemberID] = auth.OrgMember{
			MemberID: om.MemberID,
			Role:     om.Role,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) UnassignMembers(ctx context.Context, orgID string, memberIDs ...string) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, memberID := range memberIDs {
		if _, ok := orm.orgMembers[memberID]; !ok || orm.orgs[orgID].ID != orgID {
			return errors.ErrNotFound
		}
		delete(orm.orgMembers, memberID)
	}

	return nil
}

func (orm *orgRepositoryMock) UpdateMembers(ctx context.Context, oms ...auth.OrgMember) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, om := range oms {
		if _, ok := orm.orgMembers[om.MemberID]; !ok || orm.orgs[om.OrgID].ID != om.OrgID {
			return errors.ErrNotFound
		}
		orm.orgMembers[om.MemberID] = auth.OrgMember{
			MemberID: om.MemberID,
			Role:     om.Role,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	if _, ok := orm.orgMembers[memberID]; !ok {
		return "", errors.ErrNotFound
	}

	return orm.orgMembers[memberID].Role, nil
}

func (orm *orgRepositoryMock) RetrieveMembersByOrg(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	i := uint64(0)
	oms := []auth.OrgMember{}
	for _, m := range orm.orgMembers {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if _, ok := orm.orgs[orgID]; ok {
				oms = append(oms, m)
			}
		}
		i++
	}

	return auth.OrgMembersPage{
		OrgMembers: oms,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.orgMembers)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveMember(ctx context.Context, orgID, memberID string) (auth.OrgMember, error) {
	panic("not implemented")
}

func (orm *orgRepositoryMock) AssignGroups(ctx context.Context, ogs ...auth.OrgGroup) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, gr := range ogs {
		if _, ok := orm.orgs[gr.OrgID]; !ok {
			return errors.ErrNotFound
		}
		orm.orgGroups[gr.GroupID] = auth.OrgGroup{
			GroupID: gr.GroupID,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, groupID := range groupIDs {
		if _, ok := orm.orgGroups[groupID]; !ok || orm.orgs[orgID].ID != orgID {
			return errors.ErrNotFound
		}
		delete(orm.orgGroups, groupID)
	}

	return nil
}

func (orm *orgRepositoryMock) RetrieveByGroupID(ctx context.Context, groupID string) (auth.Org, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	org, ok := orm.orgGroups[groupID]
	if !ok {
		return auth.Org{}, errors.ErrNotFound
	}

	return orm.orgs[org.GroupID], nil
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

func (orm *orgRepositoryMock) RetrieveByAdmin(ctx context.Context, pm auth.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	keys := sortOrgsByID(orm.orgs)

	i := uint64(0)
	orgs := make([]auth.Org, 0)
	for _, k := range keys {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			// filter by name
			if strings.Contains(orm.orgs[k].Name, pm.Name) {
				orgs = append(orgs, orm.orgs[k])
			}
		}
		i++
	}

	return auth.OrgsPage{
		Orgs: orgs,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.orgs)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveAllMembersByOrg(ctx context.Context) ([]auth.OrgMember, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	var mrs []auth.OrgMember
	for _, org := range orm.orgs {
		for _, member := range orm.orgMembers {
			mrs = append(mrs, auth.OrgMember{
				OrgID:    org.ID,
				MemberID: member.MemberID,
			})
		}
	}

	return mrs, nil
}

func (orm *orgRepositoryMock) RetrieveAllGroupsByOrg(ctx context.Context) ([]auth.OrgGroup, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	var ogs []auth.OrgGroup
	for _, org := range orm.orgs {
		for _, group := range orm.orgGroups {
			ogs = append(ogs, auth.OrgGroup{
				OrgID:   org.ID,
				GroupID: group.GroupID,
			})
		}
	}

	return ogs, nil
}

func sortOrgsByID(orgs map[string]auth.Org) []string {
	var keys []string
	for k := range orgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
