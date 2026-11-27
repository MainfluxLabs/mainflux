// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	notifiers.Service
	things domain.ThingsClient
	pub    events.Publisher
}

// NewEventStoreMiddleware returns a notifiers service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc notifiers.Service, things domain.ThingsClient, pub events.Publisher) notifiers.Service {
	return eventStore{
		Service: svc,
		things:  things,
		pub:     pub,
	}
}

func (es eventStore) CreateNotifiers(ctx context.Context, token, groupID string, nfs ...notifiers.Notifier) ([]notifiers.Notifier, error) {
	created, err := es.Service.CreateNotifiers(ctx, token, groupID, nfs...)
	if err != nil {
		return created, err
	}

	orgIDs := es.orgIDsByGroups(ctx, created)
	for _, nf := range created {
		es.pub.Publish(ctx, events.Event{
			Action: events.NotifierCreated{
				ID:       nf.ID,
				Name:     nf.Name,
				Contacts: nf.Contacts,
				Metadata: nf.Metadata,
			},
			GroupID: nf.GroupID,
			OrgID:   orgIDs[nf.GroupID],
		})
	}

	return created, nil
}

func (es eventStore) UpdateNotifier(ctx context.Context, token string, notifier notifiers.Notifier) error {
	nf, err := es.Service.ViewNotifier(ctx, token, notifier.ID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateNotifier(ctx, token, notifier); err != nil {
		return err
	}

	orgIDs := es.orgIDsByGroups(ctx, []notifiers.Notifier{nf})
	es.pub.Publish(ctx, events.Event{
		Action: events.NotifierUpdated{
			ID:       notifier.ID,
			Name:     notifier.Name,
			Contacts: notifier.Contacts,
			Metadata: notifier.Metadata,
		},
		GroupID: nf.GroupID,
		OrgID:   orgIDs[nf.GroupID],
	})

	return nil
}

func (es eventStore) RemoveNotifiers(ctx context.Context, token string, ids ...string) ([]notifiers.Notifier, error) {
	removed, err := es.Service.RemoveNotifiers(ctx, token, ids...)
	if err != nil {
		return removed, err
	}

	orgIDs := es.orgIDsByGroups(ctx, removed)
	for _, nf := range removed {
		es.pub.Publish(ctx, events.Event{
			Action:  events.NotifierRemoved{ID: nf.ID},
			GroupID: nf.GroupID,
			OrgID:   orgIDs[nf.GroupID],
		})
	}

	return removed, nil
}

// orgIDsByGroups looks up the org of each distinct group once; a failed lookup only leaves the org out.
func (es eventStore) orgIDsByGroups(ctx context.Context, nfs []notifiers.Notifier) map[string]string {
	orgIDs := make(map[string]string)
	for _, nf := range nfs {
		if _, ok := orgIDs[nf.GroupID]; ok {
			continue
		}

		group, err := es.things.GetGroup(ctx, nf.GroupID)
		if err != nil {
			orgIDs[nf.GroupID] = ""
			continue
		}
		orgIDs[nf.GroupID] = group.OrgID
	}

	return orgIDs
}
