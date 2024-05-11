// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	contentType = "Content-Type"
	ctJSON      = "application/json"
)

var (
	ErrForward     = errors.New("failed to forward message")
	ErrSendRequest = errors.New("failed to send request")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateWebhooks creates webhooks for certain group identified by the provided ID
	CreateWebhooks(ctx context.Context, token string, webhooks ...Webhook) ([]Webhook, error)

	// ListWebhooksByGroup retrieves data about a subset of webhooks
	// related to a certain group identified by the provided ID.
	ListWebhooksByGroup(ctx context.Context, token string, thingID string) ([]Webhook, error)

	consumers.Consumer
}

type webhooksService struct {
	things     mainflux.ThingsServiceClient
	webhooks   WebhookRepository
	subscriber messaging.Subscriber
	forwarder  Forwarder
	idProvider mainflux.IDProvider
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(things mainflux.ThingsServiceClient, webhooks WebhookRepository, forwarder Forwarder, idp mainflux.IDProvider) Service {
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
	_, err := ws.things.CanAccessGroup(ctx, &mainflux.AccessGroupReq{Token: token, GroupID: webhook.GroupID, Action: things.ReadWrite})
	if err != nil {
		return Webhook{}, errors.Wrap(errors.ErrAuthorization, err)
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

func (ws *webhooksService) ListWebhooksByGroup(ctx context.Context, token string, groupID string) ([]Webhook, error) {
	_, err := ws.things.CanAccessGroup(ctx, &mainflux.AccessGroupReq{Token: token, GroupID: groupID, Action: things.Read})
	if err != nil {
		return []Webhook{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	webhooks, err := ws.webhooks.RetrieveByGroupID(ctx, groupID)
	if err != nil {
		return []Webhook{}, errors.ErrAuthorization
	}

	return webhooks, nil
}

func (ws *webhooksService) Consume(message interface{}) error {
	ctx := context.Background()

	if v, ok := message.(json.Messages); ok {
		msgs := v.Data

		for _, msg := range msgs {
			if msg.Channel == "" {
				return apiutil.ErrMissingID
			}

			//TODO: Need to get a WebhookID from Channel Profile and replace to RetrieveByID
			whs, err := ws.webhooks.RetrieveByGroupID(ctx, msg.Publisher)
			if err != nil {
				return errors.ErrAuthorization
			}

			if err := ws.forwarder.Forward(ctx, msg, whs); err != nil {
				return errors.Wrap(ErrForward, err)
			}
		}
	}

	return nil
}
