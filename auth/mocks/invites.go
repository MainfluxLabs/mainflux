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

var _ auth.OrgInvitesRepository = (*invitesRepositoryMock)(nil)

type invitesRepositoryMock struct {
	mu         sync.Mutex
	orgInvites map[string]auth.OrgInvite
}

func NewInvitesRepository() auth.OrgInvitesRepository {
	return &invitesRepositoryMock{
		orgInvites: make(map[string]auth.OrgInvite),
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

func (irm *invitesRepositoryMock) RetrieveOrgInvitesByOrgID(ctx context.Context, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
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
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(irm.orgInvites)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (irm *invitesRepositoryMock) RetrieveOrgInvitesByUserID(ctx context.Context, userType string, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
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
