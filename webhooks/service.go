// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateWebhooks creates webhooks for certain thing
	// which belongs to the user identified by a given token
	CreateWebhooks(ctx context.Context, token string, webhooks ...Webhook) ([]Webhook, error)

	// ListWebhooksByThing retrieves data about a subset of webhooks
	// related to a certain thing identified by the provided ID.
	ListWebhooksByThing(ctx context.Context, token string, thingID string) ([]Webhook, error)
}

type webhooksService struct {
	auth       mainflux.AuthServiceClient
	things     mainflux.ThingsServiceClient
	webhooks   WebhookRepository
	idProvider mainflux.IDProvider
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(auth mainflux.AuthServiceClient, things mainflux.ThingsServiceClient, webhooks WebhookRepository, idp mainflux.IDProvider) Service {
	return &webhooksService{
		auth:       auth,
		things:     things,
		webhooks:   webhooks,
		idProvider: idp,
	}
}

func (ws *webhooksService) CreateWebhooks(ctx context.Context, token string, webhooks ...Webhook) ([]Webhook, error) {
	res, err := ws.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Webhook{}, err
	}

	whs := []Webhook{}
	for _, webhook := range webhooks {
		wh, err := ws.createWebhook(ctx, &webhook, res)
		if err != nil {
			return []Webhook{}, err
		}
		whs = append(whs, wh)
	}

	return whs, nil
}

func (ws *webhooksService) createWebhook(ctx context.Context, webhook *Webhook, identity *mainflux.UserIdentity) (Webhook, error) {
	_, err := ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: identity.GetId(), ThingID: webhook.ThingID})
	if err != nil {
		if err != nil {
			return Webhook{}, errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	whs, err := ws.webhooks.Save(ctx, *webhook)
	if err != nil {
		return Webhook{}, err
	}
	if len(whs) == 0 {
		return Webhook{}, errors.ErrCreateEntity
	}
	return whs[0], nil
}

func (ws *webhooksService) ListWebhooksByThing(ctx context.Context, token string, thingID string) ([]Webhook, error) {
	res, err := ws.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Webhook{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	_, err = ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: res.GetId(), ThingID: thingID})
	if err != nil {
		if err != nil {
			return []Webhook{}, errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	webhooks, err := ws.webhooks.RetrieveByThingID(ctx, thingID)
	if err != nil {
		return []Webhook{}, err
	}

	return webhooks, nil
}
