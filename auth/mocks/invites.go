package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ auth.OrgInvitesRepository = (*invitesRepositoryMock)(nil)

type invitesRepositoryMock struct {
	mu                                sync.Mutex
	orgInvites                        map[string]auth.OrgInvite
	dormantOrgInvitesByPlatformInvite map[string][]string
}

func NewInvitesRepository() auth.OrgInvitesRepository {
	return &invitesRepositoryMock{
		orgInvites:                        make(map[string]auth.OrgInvite),
		dormantOrgInvitesByPlatformInvite: make(map[string][]string),
	}
}

func (irm *invitesRepositoryMock) SaveOrgInvite(ctx context.Context, invites ...auth.OrgInvite) error {
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

func (irm *invitesRepositoryMock) SaveDormantInviteRelation(ctx context.Context, orgInviteID, platformInviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	irm.dormantOrgInvitesByPlatformInvite[platformInviteID] = append(irm.dormantOrgInvitesByPlatformInvite[platformInviteID], orgInviteID)

	return nil
}

func (irm *invitesRepositoryMock) ActivateOrgInvite(ctx context.Context, platformInviteID, newUserID string, expiresAt time.Time) ([]auth.OrgInvite, error) {
	panic("not implemented")
}

func (irm *invitesRepositoryMock) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return auth.OrgInvite{}, dbutil.ErrNotFound
	}

	return irm.orgInvites[inviteID], nil
}

func (irm *invitesRepositoryMock) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return dbutil.ErrNotFound
	}

	delete(irm.orgInvites, inviteID)

	return nil
}

func (irm *invitesRepositoryMock) RetrieveOrgInvitesByOrg(ctx context.Context, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
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

func (irm *invitesRepositoryMock) RetrieveOrgInvitesByUser(ctx context.Context, userType, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
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
		switch userType {
		case auth.UserTypeInvitee:
			if irm.orgInvites[key].InviteeID == userID {
				invites = append(invites, irm.orgInvites[key])
			}
		case auth.UserTypeInviter:
			if irm.orgInvites[key].InviterID == userID {
				invites = append(invites, irm.orgInvites[key])
			}
		}
	}

	return auth.OrgInvitesPage{
		Invites: invites,
		Total:   uint64(len(irm.orgInvites)),
	}, nil
}

func (irm *invitesRepositoryMock) RetrieveDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (auth.OrgInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	orgInviteIDs, ok := irm.dormantOrgInvitesByPlatformInvite[platformInviteID]
	if !ok || len(orgInviteIDs) == 0 {
		return auth.OrgInvite{}, dbutil.ErrNotFound
	}

	invite, ok := irm.orgInvites[orgInviteIDs[0]]
	if !ok {
		return auth.OrgInvite{}, dbutil.ErrNotFound
	}

	return invite, nil
}

func (irm *invitesRepositoryMock) UpdateOrgInviteState(ctx context.Context, inviteID, state string) error {
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
