package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var _ mqtt.Repository = (*subRepoMock)(nil)

type subRepoMock struct {
	mu   sync.Mutex
	subs map[string][]mqtt.Subscription
}

func NewRepo(subs map[string][]mqtt.Subscription) mqtt.Repository {
	return &subRepoMock{
		subs: subs,
	}
}

func (srm *subRepoMock) RetrieveByChannelID(_ context.Context, pm mqtt.PageMetadata, chanID string) (mqtt.Page, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	i := uint64(0)

	var subs []mqtt.Subscription
	for _, s := range srm.subs {
		for _, m := range s {
			if i >= pm.Offset && i < pm.Offset+pm.Limit || pm.Limit == 0 {
				if m.ChanID == chanID {
					subs = append(subs, m)
				}
			}
			i++
		}
	}

	if len(subs) == 0 {
		return mqtt.Page{}, errors.ErrNotFound
	}

	return mqtt.Page{
		PageMetadata:  pm,
		Subscriptions: subs,
	}, nil
}

func (srm *subRepoMock) Save(_ context.Context, sub mqtt.Subscription) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, s := range srm.subs {
		for _, m := range s {
			if m.Subtopic == sub.Subtopic && m.ThingID == sub.ThingID && m.ChanID == sub.ChanID {
				return errors.ErrConflict
			}
		}
	}

	srm.subs[sub.ChanID] = append(srm.subs[sub.ChanID], sub)
	return nil
}

func (srm *subRepoMock) Remove(_ context.Context, sub mqtt.Subscription) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	for _, s := range srm.subs {
		for _, m := range s {
			if m.Subtopic == sub.Subtopic && m.ThingID == sub.ThingID && m.ChanID == sub.ChanID {
				delete(srm.subs, m.ChanID)
				return nil
			}
		}
	}

	return errors.ErrNotFound
}

func (srm *subRepoMock) UpdateStatus(_ context.Context, sub mqtt.Subscription) error {
	return nil
}

func (srm *subRepoMock) HasClientID(_ context.Context, clientID string) error {
	return nil
}
