// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
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

	//forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message interface{}) error
}

type webhooksService struct {
	auth       mainflux.AuthServiceClient
	things     mainflux.ThingsServiceClient
	webhooks   WebhookRepository
	subscriber messaging.Subscriber
	httpClient *http.Client
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(auth mainflux.AuthServiceClient, things mainflux.ThingsServiceClient, webhooks WebhookRepository) Service {
	return &webhooksService{
		auth:       auth,
		things:     things,
		webhooks:   webhooks,
		httpClient: &http.Client{},
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
	_, err := ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Owner: identity.GetId(), ThingID: webhook.ThingID})
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

	_, err = ws.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Owner: res.GetId(), ThingID: thingID})
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

// Start method starts consuming messages received from Message broker.
func Start(ctx context.Context, id string, subject string, ws Service, sub messaging.Subscriber) error {
	if err := sub.Subscribe(id, subject, handler(ctx, ws)); err != nil {
		return err
	}
	return nil
}

func handler(ctx context.Context, service Service) handleFunc {
	return func(msg messaging.Message) error {
		m := interface{}(msg)
		return service.Forward(ctx, m)
	}
}

func (ws *webhooksService) Forward(ctx context.Context, message interface{}) error {
	msg, ok := message.(messaging.Message)
	if !ok {
		return errors.New("failed to convert to Mainflux message")
	}

	whs, err := ws.webhooks.RetrieveByThingID(ctx, msg.Publisher)
	if err != nil {
		return err
	}

	for _, wh := range whs {
		err := ws.sendReq(wh.Url, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *webhooksService) sendReq(url string, msg messaging.Message) error {
	data, err := json.Marshal(msg.Payload)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ws.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Wrap(errors.New(fmt.Sprintf("error forwarding message, status : %s", resp.Status)), err)
	}
	return nil
}

type handleFunc func(msg messaging.Message) error

func (h handleFunc) Handle(msg messaging.Message) error {
	return h(msg)
}

func (h handleFunc) Cancel() error {
	return nil
}
