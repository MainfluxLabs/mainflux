// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	downlinks.Service
	things domain.ThingsClient
	pub    events.Publisher
}

// NewEventStoreMiddleware returns a downlinks service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc downlinks.Service, things domain.ThingsClient, pub events.Publisher) downlinks.Service {
	return eventStore{
		Service: svc,
		things:  things,
		pub:     pub,
	}
}

func (es eventStore) CreateDownlinks(ctx context.Context, token, thingID string, dls ...downlinks.Downlink) ([]downlinks.Downlink, error) {
	created, err := es.Service.CreateDownlinks(ctx, token, thingID, dls...)
	if err != nil {
		return created, err
	}

	orgIDs := es.orgIDsByGroups(ctx, created)
	for _, dl := range created {
		es.pub.Publish(ctx, events.Event{
			Action: events.DownlinkCreated{
				ID:       dl.ID,
				ThingID:  dl.ThingID,
				Name:     dl.Name,
				Metadata: dl.Metadata,
			},
			GroupID: dl.GroupID,
			OrgID:   orgIDs[dl.GroupID],
		})
	}

	return created, nil
}

func (es eventStore) UpdateDownlink(ctx context.Context, token string, downlink downlinks.Downlink) error {
	dl, err := es.Service.ViewDownlink(ctx, token, downlink.ID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateDownlink(ctx, token, downlink); err != nil {
		return err
	}

	orgIDs := es.orgIDsByGroups(ctx, []downlinks.Downlink{dl})
	es.pub.Publish(ctx, events.Event{
		Action: events.DownlinkUpdated{
			ID:       downlink.ID,
			Name:     downlink.Name,
			Metadata: downlink.Metadata,
		},
		GroupID: dl.GroupID,
		OrgID:   orgIDs[dl.GroupID],
	})

	return nil
}

func (es eventStore) RemoveDownlinks(ctx context.Context, token string, ids ...string) ([]downlinks.Downlink, error) {
	removed, err := es.Service.RemoveDownlinks(ctx, token, ids...)
	if err != nil {
		return removed, err
	}

	orgIDs := es.orgIDsByGroups(ctx, removed)
	for _, dl := range removed {
		es.pub.Publish(ctx, events.Event{
			Action:  events.DownlinkRemoved{ID: dl.ID},
			GroupID: dl.GroupID,
			OrgID:   orgIDs[dl.GroupID],
		})
	}

	return removed, nil
}

// orgIDsByGroups looks up the org of each distinct group once; a failed lookup only leaves the org out.
func (es eventStore) orgIDsByGroups(ctx context.Context, dls []downlinks.Downlink) map[string]string {
	orgIDs := make(map[string]string)
	for _, dl := range dls {
		if _, ok := orgIDs[dl.GroupID]; ok {
			continue
		}

		group, err := es.things.GetGroup(ctx, dl.GroupID)
		if err != nil {
			orgIDs[dl.GroupID] = ""
			continue
		}
		orgIDs[dl.GroupID] = group.OrgID
	}

	return orgIDs
}
