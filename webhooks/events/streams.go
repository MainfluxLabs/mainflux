// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

type eventStore struct {
	webhooks.Service
	things domain.ThingsClient
	pub    events.Publisher
}

// NewEventStoreMiddleware returns a webhooks service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc webhooks.Service, things domain.ThingsClient, pub events.Publisher) webhooks.Service {
	return eventStore{
		Service: svc,
		things:  things,
		pub:     pub,
	}
}

func (es eventStore) CreateWebhooks(ctx context.Context, token, thingID string, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	created, err := es.Service.CreateWebhooks(ctx, token, thingID, whs...)
	if err != nil {
		return created, err
	}

	orgIDs := es.orgIDsByGroups(ctx, created)
	for _, wh := range created {
		es.pub.Publish(ctx, events.Event{
			Action: events.WebhookCreated{
				ID:       wh.ID,
				ThingID:  wh.ThingID,
				Name:     wh.Name,
				Metadata: wh.Metadata,
			},
			GroupID: wh.GroupID,
			OrgID:   orgIDs[wh.GroupID],
		})
	}

	return created, nil
}

func (es eventStore) UpdateWebhook(ctx context.Context, token string, webhook webhooks.Webhook) error {
	wh, err := es.Service.ViewWebhook(ctx, token, webhook.ID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateWebhook(ctx, token, webhook); err != nil {
		return err
	}

	orgIDs := es.orgIDsByGroups(ctx, []webhooks.Webhook{wh})
	es.pub.Publish(ctx, events.Event{
		Action: events.WebhookUpdated{
			ID:       webhook.ID,
			Name:     webhook.Name,
			Metadata: webhook.Metadata,
		},
		GroupID: wh.GroupID,
		OrgID:   orgIDs[wh.GroupID],
	})

	return nil
}

func (es eventStore) RemoveWebhooks(ctx context.Context, token string, ids ...string) ([]webhooks.Webhook, error) {
	removed, err := es.Service.RemoveWebhooks(ctx, token, ids...)
	if err != nil {
		return removed, err
	}

	orgIDs := es.orgIDsByGroups(ctx, removed)
	for _, wh := range removed {
		es.pub.Publish(ctx, events.Event{
			Action:  events.WebhookRemoved{ID: wh.ID},
			GroupID: wh.GroupID,
			OrgID:   orgIDs[wh.GroupID],
		})
	}

	return removed, nil
}

// orgIDsByGroups looks up the org of each distinct group once; a failed lookup only leaves the org out.
func (es eventStore) orgIDsByGroups(ctx context.Context, whs []webhooks.Webhook) map[string]string {
	orgIDs := make(map[string]string)
	for _, wh := range whs {
		if _, ok := orgIDs[wh.GroupID]; ok {
			continue
		}

		group, err := es.things.GetGroup(ctx, wh.GroupID)
		if err != nil {
			orgIDs[wh.GroupID] = ""
			continue
		}
		orgIDs[wh.GroupID] = group.OrgID
	}

	return orgIDs
}
