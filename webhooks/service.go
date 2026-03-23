// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateWebhooks creates webhooks for certain thing identified by the provided ID.
	CreateWebhooks(ctx context.Context, token, thingID string, webhooks ...Webhook) ([]Webhook, error)

	// ListWebhooksByGroup retrieves data about a subset of webhooks
	// related to a certain group identified by the provided ID.
	ListWebhooksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (WebhooksPage, error)

	// ListWebhooksByThing retrieves data about a subset of webhooks
	// related to a certain thing identified by the provided ID.
	ListWebhooksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (WebhooksPage, error)

	// ViewWebhook retrieves data about the webhook identified with the provided ID.
	ViewWebhook(ctx context.Context, token, id string) (Webhook, error)

	// UpdateWebhook updates the webhook identified by the provided ID.
	UpdateWebhook(ctx context.Context, token string, webhook Webhook) error

	// RemoveWebhooks removes the webhooks identified with the provided IDs.
	RemoveWebhooks(ctx context.Context, token string, id ...string) error

	// RemoveWebhooksByThing removes webhooks related to the given thing ID.
	RemoveWebhooksByThing(ctx context.Context, thingID string) error

	// RemoveWebhooksByGroup removes webhooks related to the given group ID.
	RemoveWebhooksByGroup(ctx context.Context, groupID string) error

	consumers.Consumer
}

type webhooksService struct {
	things     domainthings.Client
	webhooks   WebhookRepository
	forwarder  Forwarder
	idProvider uuid.IDProvider
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(things domainthings.Client, webhooks WebhookRepository, forwarder Forwarder, idp uuid.IDProvider) Service {
	return &webhooksService{
		things:     things,
		webhooks:   webhooks,
		forwarder:  forwarder,
		idProvider: idp,
	}
}

func (ws *webhooksService) CreateWebhooks(ctx context.Context, token, thingID string, webhooks ...Webhook) ([]Webhook, error) {
	err := ws.things.CanUserAccessThing(ctx, domainthings.UserAccessReq{Token: token, ID: thingID, Action: domainthings.Editor})
	if err != nil {
		return []Webhook{}, err
	}

	grID, err := ws.things.GetGroupIDByThing(ctx, thingID)
	if err != nil {
		return []Webhook{}, err
	}

	whs := []Webhook{}
	for _, wh := range webhooks {
		wh.GroupID = grID
		wh.ThingID = thingID

		id, err := ws.idProvider.ID()
		if err != nil {
			return []Webhook{}, err
		}
		wh.ID = id

		whs = append(whs, wh)
	}

	whs, err = ws.webhooks.Save(ctx, whs...)
	if err != nil {
		return []Webhook{}, err
	}

	return whs, nil
}

func (ws *webhooksService) ListWebhooksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (WebhooksPage, error) {
	err := ws.things.CanUserAccessGroup(ctx, domainthings.UserAccessReq{Token: token, ID: groupID, Action: domainthings.Viewer})
	if err != nil {
		return WebhooksPage{}, err
	}

	webhooks, err := ws.webhooks.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return WebhooksPage{}, err
	}

	return webhooks, nil
}

func (ws *webhooksService) ListWebhooksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (WebhooksPage, error) {
	err := ws.things.CanUserAccessThing(ctx, domainthings.UserAccessReq{Token: token, ID: thingID, Action: domainthings.Viewer})
	if err != nil {
		return WebhooksPage{}, err
	}

	webhooks, err := ws.webhooks.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return WebhooksPage{}, err
	}

	return webhooks, nil
}

func (ws *webhooksService) ViewWebhook(ctx context.Context, token, id string) (Webhook, error) {
	webhook, err := ws.webhooks.RetrieveByID(ctx, id)
	if err != nil {
		return Webhook{}, err
	}

	if err := ws.things.CanUserAccessGroup(ctx, domainthings.UserAccessReq{Token: token, ID: webhook.GroupID, Action: domainthings.Viewer}); err != nil {
		return Webhook{}, err
	}

	return webhook, nil
}

func (ws *webhooksService) UpdateWebhook(ctx context.Context, token string, webhook Webhook) error {
	wh, err := ws.webhooks.RetrieveByID(ctx, webhook.ID)
	if err != nil {
		return err
	}

	if err := ws.things.CanUserAccessGroup(ctx, domainthings.UserAccessReq{Token: token, ID: wh.GroupID, Action: domainthings.Editor}); err != nil {
		return err
	}

	return ws.webhooks.Update(ctx, webhook)
}

func (ws *webhooksService) RemoveWebhooks(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		webhook, err := ws.webhooks.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if err := ws.things.CanUserAccessGroup(ctx, domainthings.UserAccessReq{Token: token, ID: webhook.GroupID, Action: domainthings.Editor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	if err := ws.webhooks.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func (ws *webhooksService) RemoveWebhooksByThing(ctx context.Context, thingID string) error {
	return ws.webhooks.RemoveByThing(ctx, thingID)
}

func (ws *webhooksService) RemoveWebhooksByGroup(ctx context.Context, groupID string) error {
	return ws.webhooks.RemoveByGroup(ctx, groupID)
}

func (ws *webhooksService) Consume(_ string, message any) error {
	ctx := context.Background()

	msg, ok := message.(protomfx.Message)
	if !ok {
		return errors.ErrMessage
	}

	whs, err := ws.webhooks.RetrieveByThing(ctx, msg.Publisher, apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	for _, wh := range whs.Webhooks {
		if err := ws.forwarder.Forward(ctx, msg, wh); err != nil {
			return err
		}
	}

	return nil
}
