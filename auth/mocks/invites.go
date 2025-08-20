package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ auth.InvitesRepository = (*invitesRepositoryMock)(nil)

type invitesRepositoryMock struct {
	mu              sync.Mutex
	orgInvites      map[string]auth.OrgInvite
	platformInvites map[string]auth.PlatformInvite
}

func NewInvitesRepository() auth.InvitesRepository {
	return &invitesRepositoryMock{
		orgInvites:      make(map[string]auth.OrgInvite),
		platformInvites: make(map[string]auth.PlatformInvite),
	}
}

func (irm *invitesRepositoryMock) SaveOrgInvite(ctx context.Context, invites ...auth.OrgInvite) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	for _, invite := range invites {
		if _, ok := irm.orgInvites[invite.ID]; ok {
			return errors.ErrConflict
		}

		for _, iInv := range irm.orgInvites {
			if iInv.InviteeID == invite.InviteeID &&
				iInv.OrgID == invite.OrgID &&
				iInv.InviterID == invite.InviterID &&
				iInv.State == "pending" &&
				iInv.ExpiresAt.After(time.Now()) {
				return errors.ErrConflict
			}
		}

		irm.orgInvites[invite.ID] = invite
	}

	return nil
}

func (irm *invitesRepositoryMock) RetrieveOrgInviteByID(ctx context.Context, inviteID string) (auth.OrgInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return auth.OrgInvite{}, errors.ErrNotFound
	}

	return irm.orgInvites[inviteID], nil
}

func (irm *invitesRepositoryMock) RemoveOrgInvite(ctx context.Context, inviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return errors.ErrNotFound
	}

	delete(irm.orgInvites, inviteID)

	return nil
}

func (irm *invitesRepositoryMock) RetrieveOrgInvitesByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (auth.OrgInvitesPage, error) {
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
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(irm.orgInvites)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (irm *invitesRepositoryMock) UpdateOrgInviteState(ctx context.Context, inviteID string, state string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.orgInvites[inviteID]; !ok {
		return errors.ErrNotFound
	}

	inv := irm.orgInvites[inviteID]
	inv.State = state

	irm.orgInvites[inviteID] = inv
	return nil
}

func (irm *invitesRepositoryMock) SavePlatformInvite(ctx context.Context, invites ...auth.PlatformInvite) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	for _, invite := range invites {
		if _, ok := irm.platformInvites[invite.ID]; ok {
			return errors.ErrConflict
		}

		for _, iInv := range irm.platformInvites {
			if iInv.InviteeEmail == invite.InviteeEmail &&
				iInv.State == "pending" &&
				iInv.ExpiresAt.After(time.Now()) {
				return errors.ErrConflict
			}
		}

		irm.platformInvites[invite.ID] = invite
	}

	return nil
}

func (irm *invitesRepositoryMock) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (auth.PlatformInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.platformInvites[inviteID]; !ok {
		return auth.PlatformInvite{}, errors.ErrNotFound
	}

	return irm.platformInvites[inviteID], nil
}

func (irm *invitesRepositoryMock) RetrievePlatformInvites(ctx context.Context, pm apiutil.PageMetadata) (auth.PlatformInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.platformInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]auth.PlatformInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		invites = append(invites, irm.platformInvites[key])
	}

	return auth.PlatformInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(irm.platformInvites)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (irm *invitesRepositoryMock) UpdatePlatformInviteState(ctx context.Context, inviteID string, state string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.platformInvites[inviteID]; !ok {
		return errors.ErrNotFound
	}

	inv := irm.platformInvites[inviteID]
	inv.State = state

	irm.platformInvites[inviteID] = inv
	return nil
}
