package mocks

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Service = (*mainfluxWebhooks)(nil)

type mainfluxWebhooks struct {
	mu       sync.Mutex
	things   mainflux.ThingsServiceClient
	webhooks map[string]webhooks.Webhook
}

func (svc *mainfluxWebhooks) CreateWebhooks(ctx context.Context, token string, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	for i := range whs {
		_, err := svc.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: token, ThingID: whs[i].ThingID})
		if err != nil {
			return []webhooks.Webhook{}, errors.ErrAuthorization
		}

		svc.webhooks[whs[i].ThingID] = whs[i]
	}

	return whs, nil
}

func (svc *mainfluxWebhooks) ListWebhooksByThing(ctx context.Context, token string, thingID string) ([]webhooks.Webhook, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	_, err := svc.things.IsThingOwner(ctx, &mainflux.ThingOwnerReq{Token: token, ThingID: thingID})
	if err != nil {
		return nil, errors.ErrAuthorization
	}

	var whs []webhooks.Webhook

	for _, webhook := range svc.webhooks {
		if webhook.ThingID == thingID {
			whs = append(whs, webhook)
		}
	}

	return whs, nil
}

func (svc *mainfluxWebhooks) Forward(message messaging.Message) error {
	panic("not implemented")
}

func (svc *mainfluxWebhooks) Consume(messages interface{}) error {
	panic("not implemented")
}
