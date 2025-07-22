package mocks

import (
	"context"
	"sort"
	"sync"

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
			if iInv.InviteeID == invite.InviteeID && iInv.OrgID == invite.OrgID {
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

func (irm *invitesRepositoryMock) RetrieveByInviteeID(ctx context.Context, inviteeID string, pm apiutil.PageMetadata) (auth.InvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.invites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]auth.Invite, 0)
	for _, key := range keys[pm.Offset : pm.Offset+pm.Limit] {
		if irm.invites[key].InviteeID == inviteeID {
			invites = append(invites, irm.invites[key])
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
