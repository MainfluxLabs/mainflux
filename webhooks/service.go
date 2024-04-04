// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"bytes"
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

const (
	contentType = "Content-Type"
	ctJSON      = "application/json"
)

var (
	ErrForward     = errors.New("Error forwarding message")
	ErrMessage     = errors.New("Failed to convert to Mainflux message")
	ErrSendRequest = errors.New("Error sending request")
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

	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message messaging.Message) error

	consumers.Consumer
}

type webhooksService struct {
	things     mainflux.ThingsServiceClient
	webhooks   WebhookRepository
	subscriber messaging.Subscriber
	httpClient *http.Client
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(things mainflux.ThingsServiceClient, webhooks WebhookRepository) Service {
	return &webhooksService{
		things:     things,
		webhooks:   webhooks,
		httpClient: &http.Client{},
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
	_, err := ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: token, ThingID: webhook.ThingID})
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
	_, err := ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: token, ThingID: thingID})
	if err != nil {
		if err != nil {
			return []Webhook{}, errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	webhooks, err := ws.webhooks.RetrieveByThingID(ctx, thingID)
	if err != nil {
		return []Webhook{}, errors.ErrAuthorization
	}

	return webhooks, nil
}

func (ws *webhooksService) Forward(ctx context.Context, msg messaging.Message) error {
	if msg.Publisher == "" {
		return apiutil.ErrMissingID
	}

	whs, err := ws.webhooks.RetrieveByThingID(ctx, msg.Publisher)
	if err != nil {
		return errors.ErrAuthorization
	}

	for _, wh := range whs {
		if err := ws.sendRequest(wh.Url, msg); err != nil {
			return err
		}
	}

	return nil
}

func (ws *webhooksService) sendRequest(url string, msg messaging.Message) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg.Payload))
	if err != nil {
		return err
	}

	req.Header.Set(contentType, ctJSON)

	resp, err := ws.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(ErrSendRequest, err)
	}
	defer resp.Body.Close()

	return nil
}

func (ws *webhooksService) Consume(message interface{}) error {
	ctx := context.Background()

	msg, ok := message.(messaging.Message)
	if !ok {
		return ErrMessage
	}

	err := ws.Forward(ctx, msg)
	if err != nil {
		return errors.Wrap(ErrForward, err)
	}

	return nil
}
