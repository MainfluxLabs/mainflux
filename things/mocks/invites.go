package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ things.GroupInviteRepository = (*invitesRepositoryMock)(nil)

type invitesRepositoryMock struct {
	mu                             sync.Mutex
	groupInvites                   map[string]things.GroupInvite
	dormantGroupInvitesByOrgInvite map[string][]string
}

func NewInvitesRepository() things.GroupInviteRepository {
	return &invitesRepositoryMock{
		groupInvites:                   make(map[string]things.GroupInvite),
		dormantGroupInvitesByOrgInvite: make(map[string][]string),
	}
}

func (irm *invitesRepositoryMock) SaveInvites(ctx context.Context, invites ...things.GroupInvite) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	for _, invite := range invites {
		if _, ok := irm.groupInvites[invite.ID]; ok {
			return dbutil.ErrConflict
		}

		for _, existingInvite := range irm.groupInvites {
			if existingInvite.InviteeID == invite.InviteeID &&
				existingInvite.GroupID == invite.GroupID &&
				existingInvite.InviterID == invite.InviterID &&
				existingInvite.State == "pending" &&
				existingInvite.ExpiresAt.After(time.Now()) {
				return dbutil.ErrConflict
			}
		}

		irm.groupInvites[invite.ID] = invite
	}

	return nil
}

func (irm *invitesRepositoryMock) SaveDormantInviteRelations(ctx context.Context, orgInviteID string, groupInviteIDs ...string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	irm.dormantGroupInvitesByOrgInvite[orgInviteID] = append(irm.dormantGroupInvitesByOrgInvite[orgInviteID], orgInviteID)

	return nil
}

func (irm *invitesRepositoryMock) ActivateGroupInvites(ctx context.Context, orgInviteID, userID string, expirationTime time.Time) ([]things.GroupInvite, error) {
	panic("not implemented")
}

func (irm *invitesRepositoryMock) RetrieveInviteByID(ctx context.Context, inviteID string) (things.GroupInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.groupInvites[inviteID]; !ok {
		return things.GroupInvite{}, dbutil.ErrNotFound
	}

	return irm.groupInvites[inviteID], nil
}

func (irm *invitesRepositoryMock) RemoveInvite(ctx context.Context, inviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.groupInvites[inviteID]; !ok {
		return dbutil.ErrNotFound
	}

	delete(irm.groupInvites, inviteID)

	return nil
}

func (irm *invitesRepositoryMock) RetrieveInvitesByDestination(ctx context.Context, groupID string, pm invites.PageMetadataInvites) (things.GroupInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.groupInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]things.GroupInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		if irm.groupInvites[key].GroupID == groupID {
			invites = append(invites, irm.groupInvites[key])
		}
	}

	return things.GroupInvitesPage{
		Invites: invites,
		Total:   uint64(len(irm.groupInvites)),
	}, nil
}

func (irm *invitesRepositoryMock) RetrieveInvitesByUser(ctx context.Context, userType, userID string, pm invites.PageMetadataInvites) (things.GroupInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.groupInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	retInvites := make([]things.GroupInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		switch userType {
		case invites.UserTypeInvitee:
			if irm.groupInvites[key].InviteeID.String == userID {
				retInvites = append(retInvites, irm.groupInvites[key])
			}
		case invites.UserTypeInviter:
			if irm.groupInvites[key].InviterID == userID {
				retInvites = append(retInvites, irm.groupInvites[key])
			}
		}
	}

	return things.GroupInvitesPage{
		Invites: retInvites,
		Total:   uint64(len(irm.groupInvites)),
	}, nil
}

func (irm *invitesRepositoryMock) UpdateInviteState(ctx context.Context, inviteID, state string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.groupInvites[inviteID]; !ok {
		return dbutil.ErrNotFound
	}

	inv := irm.groupInvites[inviteID]
	inv.State = state

	irm.groupInvites[inviteID] = inv
	return nil
}
