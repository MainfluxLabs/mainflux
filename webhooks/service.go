// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

var ErrForward = errors.New("failed to forward message")

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateWebhooks creates webhooks for certain group identified by the provided ID
	CreateWebhooks(ctx context.Context, token string, webhooks ...Webhook) ([]Webhook, error)

	// ListWebhooksByGroup retrieves data about a subset of webhooks
	// related to a certain group identified by the provided ID.
	ListWebhooksByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (WebhooksPage, error)

	// ViewWebhook retrieves data about the webhook identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewWebhook(ctx context.Context, token, id string) (Webhook, error)

	// UpdateWebhook updates the webhook identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateWebhook(ctx context.Context, token string, webhook Webhook) error

	// RemoveWebhooks removes the webhooks identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveWebhooks(ctx context.Context, token string, id ...string) error

	consumers.Consumer
}

type PageMetadata struct {
	Total    uint64
	Offset   uint64                 `json:"offset,omitempty"`
	Limit    uint64                 `json:"limit,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Order    string                 `json:"order,omitempty"`
	Dir      string                 `json:"dir,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type webhooksService struct {
	things     protomfx.ThingsServiceClient
	webhooks   WebhookRepository
	subscriber messaging.Subscriber
	forwarder  Forwarder
	idProvider uuid.IDProvider
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(things protomfx.ThingsServiceClient, webhooks WebhookRepository, forwarder Forwarder, idp uuid.IDProvider) Service {
	return &webhooksService{
		things:     things,
		webhooks:   webhooks,
		forwarder:  forwarder,
		idProvider: idp,
	}
}

func (ws *webhooksService) CreateWebhooks(ctx context.Context, token string, webhooks ...Webhook) ([]Webhook, error) {
	whs := []Webhook{}
	for _, webhook := range webhooks {
		wh, err := ws.createWebhook(ctx, &webhook, token)
		if err != nil {
			return []Webhook{}, err
		}
		whs = append(whs, wh)
	}

	return whs, nil
}

func (ws *webhooksService) createWebhook(ctx context.Context, webhook *Webhook, token string) (Webhook, error) {
	_, err := ws.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: webhook.GroupID, Action: things.Editor})
	if err != nil {
		return Webhook{}, err
	}

	id, err := ws.idProvider.ID()
	if err != nil {
		return Webhook{}, err
	}
	webhook.ID = id

	whs, err := ws.webhooks.Save(ctx, *webhook)
	if err != nil {
		return Webhook{}, err
	}

	if len(whs) == 0 {
		return Webhook{}, errors.ErrCreateEntity
	}

	return whs[0], nil
}

func (ws *webhooksService) ListWebhooksByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (WebhooksPage, error) {
	_, err := ws.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return WebhooksPage{}, err
	}

	webhooks, err := ws.webhooks.RetrieveByGroupID(ctx, groupID, pm)
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

	if _, err := ws.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: webhook.GroupID, Action: things.Viewer}); err != nil {
		return Webhook{}, err
	}

	return webhook, nil
}

func (ws *webhooksService) UpdateWebhook(ctx context.Context, token string, webhook Webhook) error {
	wh, err := ws.webhooks.RetrieveByID(ctx, webhook.ID)
	if err != nil {
		return err
	}

	if _, err := ws.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: wh.GroupID, Action: things.Editor}); err != nil {
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
		if _, err := ws.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: webhook.GroupID, Action: things.Editor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	if err := ws.webhooks.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func (ws *webhooksService) Consume(message interface{}) error {
	ctx := context.Background()

	if v, ok := message.(json.Messages); ok {
		msgs := v.Data
		for _, msg := range msgs {
			if msg.ProfileConfig["webhook_id"] == nil {
				return apiutil.ErrMissingID
			}

			wh, err := ws.webhooks.RetrieveByID(ctx, msg.ProfileConfig["webhook_id"].(string))
			if err != nil {
				return err
			}

			if err := ws.forwarder.Forward(ctx, msg, wh); err != nil {
				return errors.Wrap(ErrForward, err)
			}
		}
	}

	return nil
}
