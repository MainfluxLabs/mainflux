package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.WebhookRepository = (*webhookRepositoryMock)(nil)

type webhookRepositoryMock struct {
	mu       sync.Mutex
	webhooks map[string]webhooks.Webhook
}

func NewWebhookRepository() webhooks.WebhookRepository {
	return &webhookRepositoryMock{
		webhooks: make(map[string]webhooks.Webhook),
	}
}

func (wrm *webhookRepositoryMock) Save(_ context.Context, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()

	for _, wh := range whs {
		for _, w := range wrm.webhooks {
			if w.GroupID == wh.GroupID && w.Name == wh.Name {
				return []webhooks.Webhook{}, errors.ErrConflict
			}
		}

		wrm.webhooks[wh.ID] = wh
	}

	return whs, nil
}

func (wrm *webhookRepositoryMock) RetrieveByGroupID(_ context.Context, groupID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()
	var items []webhooks.Webhook

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	for _, wh := range wrm.webhooks {
		if wh.GroupID == groupID {
			id := uuid.ParseID(wh.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, wh)
			}
		}
	}

	return webhooks.WebhooksPage{
		Webhooks: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(items)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (wrm *webhookRepositoryMock) RetrieveByThingID(_ context.Context, thingID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()
	var items []webhooks.Webhook

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	for _, wh := range wrm.webhooks {
		if wh.ThingID == thingID {
			id := uuid.ParseID(wh.ID)
			if id >= first && id < last || pm.Limit == 0 {
				items = append(items, wh)
			}
		}
	}

	return webhooks.WebhooksPage{
		Webhooks: items,
		PageMetadata: apiutil.PageMetadata{
			Total:  uint64(len(items)),
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (wrm *webhookRepositoryMock) RetrieveByID(_ context.Context, id string) (webhooks.Webhook, error) {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()

	for _, wh := range wrm.webhooks {
		if wh.ID == id {
			return wh, nil
		}
	}

	return webhooks.Webhook{}, errors.ErrNotFound
}

func (wrm *webhookRepositoryMock) Update(_ context.Context, w webhooks.Webhook) error {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()

	if _, ok := wrm.webhooks[w.ID]; !ok {
		return errors.ErrNotFound
	}
	wrm.webhooks[w.ID] = w

	return nil
}

func (wrm *webhookRepositoryMock) Remove(_ context.Context, ids ...string) error {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()

	for _, id := range ids {
		if _, ok := wrm.webhooks[id]; !ok {
			return errors.ErrNotFound
		}
		delete(wrm.webhooks, id)
	}

	return nil
}
