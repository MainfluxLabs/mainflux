package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.PlatformInvitesRepository = (*platformInvitesRepositoryMock)(nil)

type platformInvitesRepositoryMock struct {
	mu              sync.Mutex
	platformInvites map[string]users.PlatformInvite
}

func NewPlatformInvitesRepository() users.PlatformInvitesRepository {
	return &platformInvitesRepositoryMock{
		platformInvites: make(map[string]users.PlatformInvite),
	}
}

func (irm *platformInvitesRepositoryMock) SavePlatformInvite(ctx context.Context, invites ...users.PlatformInvite) error {
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

func (irm *platformInvitesRepositoryMock) RetrievePlatformInviteByID(ctx context.Context, inviteID string) (users.PlatformInvite, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	if _, ok := irm.platformInvites[inviteID]; !ok {
		return users.PlatformInvite{}, errors.ErrNotFound
	}

	return irm.platformInvites[inviteID], nil
}

func (irm *platformInvitesRepositoryMock) RetrievePlatformInvites(ctx context.Context, pm apiutil.PageMetadata) (users.PlatformInvitesPage, error) {
	irm.mu.Lock()
	defer irm.mu.Unlock()

	keys := make([]string, 0)
	for k := range irm.platformInvites {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	invites := make([]users.PlatformInvite, 0)
	idxEnd := pm.Offset + pm.Limit
	if idxEnd > uint64(len(keys)) {
		idxEnd = uint64(len(keys))
	}

	for _, key := range keys[pm.Offset:idxEnd] {
		invites = append(invites, irm.platformInvites[key])
	}

	return users.PlatformInvitesPage{
		Invites: invites,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(irm.platformInvites)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (irm *platformInvitesRepositoryMock) UpdatePlatformInviteState(ctx context.Context, inviteID string, state string) error {
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
