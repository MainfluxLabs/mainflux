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
	mu           sync.Mutex
	orgs         map[string]auth.Org
	members      map[string]auth.OrgMember
	groups       map[string]auth.Group
	groupMembers map[string]auth.GroupInvitationByID
}

// NewOrgRepository returns mock of org repository
func NewOrgRepository() auth.OrgRepository {
	return &orgRepositoryMock{
		orgs:         make(map[string]auth.Org),
		members:      make(map[string]auth.OrgMember),
		groups:       make(map[string]auth.Group),
		groupMembers: make(map[string]auth.GroupInvitationByID),
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

func (orm *orgRepositoryMock) RetrieveMemberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.OrgsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	i := uint64(0)
	orgs := make([]auth.Org, 0)
	for _, org := range orm.orgs {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if _, ok := orm.members[memberID]; ok {
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
		orm.members[om.MemberID] = auth.OrgMember{
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
		if _, ok := orm.members[memberID]; !ok || orm.orgs[orgID].ID != orgID {
			return errors.ErrNotFound
		}
		delete(orm.members, memberID)
	}

	return nil
}

func (orm *orgRepositoryMock) UpdateMembers(ctx context.Context, oms ...auth.OrgMember) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, om := range oms {
		if _, ok := orm.members[om.MemberID]; !ok || orm.orgs[om.OrgID].ID != om.OrgID {
			return errors.ErrNotFound
		}
		orm.members[om.MemberID] = auth.OrgMember{
			MemberID: om.MemberID,
			Role:     om.Role,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) RetrieveRole(ctx context.Context, memberID, orgID string) (string, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	if _, ok := orm.members[memberID]; !ok {
		return "", errors.ErrNotFound
	}

	return orm.members[memberID].Role, nil
}

func (orm *orgRepositoryMock) RetrieveMembers(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgMembersPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	i := uint64(0)
	members := []auth.OrgMember{}
	for _, member := range orm.members {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if _, ok := orm.orgs[orgID]; ok {
				members = append(members, member)
			}
		}
		i++
	}

	return auth.OrgMembersPage{
		Members: members,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.members)),
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
		orm.groups[gr.GroupID] = auth.Group{
			ID: gr.GroupID,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, groupID := range groupIDs {
		if _, ok := orm.groups[groupID]; !ok || orm.orgs[orgID].ID != orgID {
			return errors.ErrNotFound
		}
		delete(orm.groups, groupID)
	}

	return nil
}

func (orm *orgRepositoryMock) RetrieveGroups(ctx context.Context, orgID string, pm auth.PageMetadata) (auth.OrgGroupsPage, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	i := uint64(0)
	ogs := []auth.OrgGroup{}
	for _, group := range orm.groups {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			if _, ok := orm.orgs[orgID]; ok {
				ogs = append(ogs, auth.OrgGroup{
					OrgID:   orgID,
					GroupID: group.ID,
				})
			}
		}
		i++
	}

	return auth.OrgGroupsPage{
		OrgGroups: ogs,
		PageMetadata: auth.PageMetadata{
			Total:  uint64(len(orm.groups)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (orm *orgRepositoryMock) RetrieveByGroupID(ctx context.Context, groupID string) (auth.Org, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	org, ok := orm.groups[groupID]
	if !ok {
		return auth.Org{}, errors.ErrNotFound
	}

	return orm.orgs[org.ID], nil
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

func (orm *orgRepositoryMock) RetrieveAllOrgMembers(ctx context.Context) ([]auth.OrgMember, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	var mrs []auth.OrgMember
	for _, org := range orm.orgs {
		for _, member := range orm.members {
			mrs = append(mrs, auth.OrgMember{
				OrgID:    org.ID,
				MemberID: member.MemberID,
			})
		}
	}

	return mrs, nil
}

func (orm *orgRepositoryMock) RetrieveAllOrgGroups(ctx context.Context) ([]auth.OrgGroup, error) {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	var ogs []auth.OrgGroup
	for _, org := range orm.orgs {
		for _, group := range orm.groups {
			ogs = append(ogs, auth.OrgGroup{
				OrgID:   org.ID,
				GroupID: group.ID,
			})
		}
	}

	return ogs, nil
}

func (orm *orgRepositoryMock) SaveGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	orm.mu.Lock()
	defer orm.mu.Unlock()

	for _, g := range giByIDs {
		if _, ok := orm.members[g.MemberID]; !ok {
			return errors.ErrNotFound
		}

		orm.groupMembers[g.MemberID] = auth.GroupInvitationByID{
			MemberID: g.MemberID,
			Policy:   g.Policy,
		}
	}

	return nil
}

func (orm *orgRepositoryMock) RetrieveGroupMember(ctx context.Context, gp auth.GroupsPolicy) (string, error) {
	panic("not implemented")
}

func (orm *orgRepositoryMock) RetrieveGroupMembers(ctx context.Context, groupID string, pm auth.PageMetadata) (auth.GroupMembersPage, error) {
	panic("not implemented")
}

func (orm *orgRepositoryMock) UpdateGroupMembers(ctx context.Context, groupID string, giByIDs ...auth.GroupInvitationByID) error {
	panic("not implemented")
}

func (orm *orgRepositoryMock) RemoveGroupMembers(ctx context.Context, groupID string, memberIDs ...string) error {
	panic("not implemented")
}

func sortOrgsByID(orgs map[string]auth.Org) []string {
	var keys []string
	for k := range orgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
