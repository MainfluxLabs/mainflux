// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"

	//"errors"
	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	CreateWebhook(ctx context.Context, token string, webhook Webhook) (Webhook, error)
	ListWebhooksByThing(ctx context.Context, token string, thingID string) ([]Webhook, error)
}

type webhooksService struct {
	auth       mainflux.AuthServiceClient
	webhooks   WebhookRepository
	idProvider mainflux.IDProvider
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(auth mainflux.AuthServiceClient, webhooks WebhookRepository, idp mainflux.IDProvider) Service {
	return &webhooksService{
		auth:       auth,
		webhooks:   webhooks,
		idProvider: idp,
	}
}

func (ws *webhooksService) CreateWebhook(ctx context.Context, token string, webhook Webhook) (Webhook, error) {
	_, err := ws.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Webhook{}, err
	}

	wh, err := ws.webhooks.Save(ctx, webhook)
	if err != nil {
		return Webhook{}, err
	}
	return wh, nil
}

func (ws *webhooksService) ListWebhooksByThing(ctx context.Context, token string, thingID string) ([]Webhook, error) {
	_, err := ws.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Webhook{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	webhooks, err := ws.webhooks.RetrieveByThingID(ctx, thingID)
	if err != nil {
		return []Webhook{}, err
	}
	return webhooks, nil
}
