package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
)

var _ auth.OrgInviteRepository = (*invitesRepositoryMock)(nil)

type invitesRepositoryMock struct {
	mu         sync.Mutex
	orgInvites map[string]auth.OrgInvite
}

func NewInvitesRepository() auth.OrgInviteRepository {
	return &invitesRepositoryMock{
		orgInvites: make(map[string]auth.OrgInvite),
	}
}

func (irm *invitesRepositoryMock) SaveInvites(ctx context.Context, invites ...auth.OrgInvite) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	for _, invite := range invites {
		if _, ok := irm.orgInvites[invite.ID]; ok {
			return dbutil.ErrConflict
		}

		for _, existingInvite := range irm.orgInvites {
			if existingInvite.InviteeID == invite.InviteeID &&
				existingInvite.OrgID == invite.OrgID &&
				existingInvite.InviterID == invite.InviterID &&
				existingInvite.State == "pending" &&
				existingInvite.ExpiresAt.After(time.Now()) {
				return dbutil.ErrConflict
			}
		}

		irm.orgInvites[invite.ID] = invite
	}

	return nil
}

func (irm *invitesRepositoryMock) RetrieveInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return auth.OrgInvite{}, dbutil.ErrNotFound
	}

	return irm.orgInvites[inviteID], nil
}

func (irm *invitesRepositoryMock) RemoveInvite(ctx context.Context, inviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return dbutil.ErrNotFound
	}

	delete(irm.orgInvites, inviteID)

	return nil
}

func (irm *invitesRepositoryMock) RetrieveInvitesByDestination(ctx context.Context, orgID string, pm invites.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.orgInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]auth.OrgInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		if irm.orgInvites[key].OrgID == orgID {
			invites = append(invites, irm.orgInvites[key])
		}
	}

	return auth.OrgInvitesPage{
		Invites: invites,
		Total:   uint64(len(irm.orgInvites)),
	}, nil
}

func (irm *invitesRepositoryMock) RetrieveInvitesByUser(ctx context.Context, userType, userID string, pm invites.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.orgInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	retInvites := make([]auth.OrgInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		switch userType {
		case invites.UserTypeInvitee:
			if irm.orgInvites[key].InviteeID.String == userID {
				retInvites = append(retInvites, irm.orgInvites[key])
			}
		case invites.UserTypeInviter:
			if irm.orgInvites[key].InviterID == userID {
				retInvites = append(retInvites, irm.orgInvites[key])
			}
		}
	}

	return auth.OrgInvitesPage{
		Invites: retInvites,
		Total:   uint64(len(irm.orgInvites)),
	}, nil
}

func (irm *invitesRepositoryMock) UpdateInviteState(ctx context.Context, inviteID, state string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return dbutil.ErrNotFound
	}

	inv := irm.orgInvites[inviteID]
	inv.State = state

	irm.orgInvites[inviteID] = inv
	return nil
}
