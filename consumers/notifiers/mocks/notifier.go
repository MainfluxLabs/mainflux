// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ notifiers.Notifier = (*notifier)(nil)
var _ notifiers.NotifierRepository = (*notifierRepositoryMock)(nil)

const invalidSender = "invalid@example.com"

type notifier struct{}

// NewNotifier returns a new Notifier mock.
func NewNotifier() notifiers.Notifier {
	return notifier{}
}

type notifierRepositoryMock struct {
	mu        sync.Mutex
	notifiers map[string]things.Notifier
}

func NewNotifierRepository() notifiers.NotifierRepository {
	return &notifierRepositoryMock{notifiers: make(map[string]things.Notifier)}
}

func (n notifier) Notify(from string, to []string, msg protomfx.Message) error {
	if len(to) < 1 {
		return notifiers.ErrNotify
	}

	for _, t := range to {
		if t == invalidSender || t == "" {
			return notifiers.ErrNotify
		}
	}

	return nil
}

func (nrm *notifierRepositoryMock) Save(_ context.Context, nfs ...things.Notifier) ([]things.Notifier, error) {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	for i := range nfs {
		nrm.notifiers[nfs[i].ID] = nfs[i]
	}
	return nfs, nil
}

func (nrm *notifierRepositoryMock) RetrieveByGroupID(_ context.Context, groupID string) ([]things.Notifier, error) {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	var nfs []things.Notifier
	for _, i := range nrm.notifiers {
		if i.GroupID == groupID {
			nfs = append(nfs, i)
		}
	}

	return nfs, nil
}

func (nrm *notifierRepositoryMock) RetrieveByID(_ context.Context, id string) (things.Notifier, error) {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	for _, nf := range nrm.notifiers {
		if nf.ID == id {
			return nf, nil
		}
	}

	return things.Notifier{}, errors.ErrNotFound
}

func (nrm *notifierRepositoryMock) Update(_ context.Context, nf things.Notifier) error {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	if _, ok := nrm.notifiers[nf.ID]; !ok {
		return errors.ErrNotFound
	}
	nrm.notifiers[nf.ID] = nf

	return nil
}

func (nrm *notifierRepositoryMock) Remove(_ context.Context, groupID string, ids ...string) error {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	for _, id := range ids {
		if _, ok := nrm.notifiers[id]; !ok {
			return errors.ErrNotFound
		}
		delete(nrm.notifiers, id)
	}

	return nil
}
