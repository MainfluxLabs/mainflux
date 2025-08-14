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
	mu      sync.Mutex
	invites map[string]auth.Invite
}

func NewInvitesRepository() auth.InvitesRepository {
	return &invitesRepositoryMock{
		invites: make(map[string]auth.Invite),
	}
}

func (irm *invitesRepositoryMock) Save(ctx context.Context, invites ...auth.Invite) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	for _, invite := range invites {
		if _, ok := irm.invites[invite.ID]; ok {
			return errors.ErrConflict
		}

		for _, iInv := range irm.invites {
			if iInv.InviteeID != "" &&
				iInv.InviteeID == invite.InviteeID &&
				iInv.OrgID == invite.OrgID &&
				iInv.InviterID == invite.InviterID &&
				iInv.ExpiresAt.After(time.Now()) {
				return errors.ErrConflict
			}

			if iInv.InviteeEmail != "" &&
				iInv.InviteeEmail == invite.InviteeEmail &&
				iInv.OrgID == invite.OrgID &&
				iInv.InviterID == invite.InviterID &&
				iInv.ExpiresAt.After(time.Now()) {
				return errors.ErrConflict
			}
		}

		irm.invites[invite.ID] = invite
	}

	return nil
}

func (irm *invitesRepositoryMock) RetrieveByID(ctx context.Context, inviteID string) (auth.Invite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.invites[inviteID]; !ok {
		return auth.Invite{}, errors.ErrNotFound
	}

	return irm.invites[inviteID], nil
}

func (irm *invitesRepositoryMock) Remove(ctx context.Context, inviteID string) error {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.invites[inviteID]; !ok {
		return errors.ErrNotFound
	}

	delete(irm.invites, inviteID)

	return nil
}

func (irm *invitesRepositoryMock) RetrieveByUserID(ctx context.Context, userType string, userID string, pm apiutil.PageMetadata) (auth.InvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.invites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]auth.Invite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		switch userType {
		case auth.UserTypeInvitee:
			if irm.invites[key].InviteeID == userID {
				invites = append(invites, irm.invites[key])
			}
		case auth.UserTypeInviter:
			if irm.invites[key].InviterID == userID {
				invites = append(invites, irm.invites[key])
			}
		}
	}

	return auth.InvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(irm.invites)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (irm *invitesRepositoryMock) FlipInactiveInvites(ctx context.Context, email string, inviteeID string) (uint32, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	cnt := uint32(0)

	for inviteID, inv := range irm.invites {
		if inv.InviteeEmail == email {
			inv.InviteeID = inviteeID
			inv.InviteeEmail = ""

			irm.invites[inviteID] = inv

			cnt += 1
		}
	}

	return cnt, nil
}
