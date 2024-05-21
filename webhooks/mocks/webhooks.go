package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.WebhookRepository = (*webhookRepositoryMock)(nil)

type webhookRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
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

	for i := range whs {
		wrm.webhooks[whs[i].ID] = whs[i]
	}
	return whs, nil
}

func (wrm *webhookRepositoryMock) RetrieveByGroupID(_ context.Context, groupID string) ([]webhooks.Webhook, error) {
	wrm.mu.Lock()
	defer wrm.mu.Unlock()

	var whs []webhooks.Webhook
	for _, i := range wrm.webhooks {
		if i.GroupID == groupID {
			whs = append(whs, i)
		}
	}

	return whs, nil
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

func (wrm *webhookRepositoryMock) Remove(_ context.Context, groupID string, ids ...string) error {
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
