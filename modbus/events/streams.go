// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	modbus.Service
	things domain.ThingsClient
	pub    events.Publisher
}

// NewEventStoreMiddleware returns a modbus service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc modbus.Service, things domain.ThingsClient, pub events.Publisher) modbus.Service {
	return eventStore{
		Service: svc,
		things:  things,
		pub:     pub,
	}
}

func (es eventStore) CreateClients(ctx context.Context, token, thingID string, clients ...modbus.Client) ([]modbus.Client, error) {
	created, err := es.Service.CreateClients(ctx, token, thingID, clients...)
	if err != nil {
		return created, err
	}

	es.publishCreated(ctx, created...)

	return created, nil
}

func (es eventStore) CreateClient(ctx context.Context, token, thingID string, client modbus.Client) (modbus.Client, error) {
	created, err := es.Service.CreateClient(ctx, token, thingID, client)
	if err != nil {
		return created, err
	}

	es.publishCreated(ctx, created)

	return created, nil
}

func (es eventStore) UpdateClient(ctx context.Context, token string, client modbus.Client) error {
	cl, err := es.Service.ViewClient(ctx, token, client.ID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateClient(ctx, token, client); err != nil {
		return err
	}

	orgIDs := es.orgIDsByGroups(ctx, []modbus.Client{cl})
	es.pub.Publish(ctx, events.Event{
		Action: events.ModbusClientUpdated{
			ID:       client.ID,
			Name:     client.Name,
			Metadata: client.Metadata,
		},
		GroupID: cl.GroupID,
		OrgID:   orgIDs[cl.GroupID],
	})

	return nil
}

func (es eventStore) RemoveClients(ctx context.Context, token string, ids ...string) ([]modbus.Client, error) {
	removed, err := es.Service.RemoveClients(ctx, token, ids...)
	if err != nil {
		return removed, err
	}

	orgIDs := es.orgIDsByGroups(ctx, removed)
	for _, cl := range removed {
		es.pub.Publish(ctx, events.Event{
			Action:  events.ModbusClientRemoved{ID: cl.ID},
			GroupID: cl.GroupID,
			OrgID:   orgIDs[cl.GroupID],
		})
	}

	return removed, nil
}

// publishCreated publishes a creation event for each of the clients.
func (es eventStore) publishCreated(ctx context.Context, cls ...modbus.Client) {
	orgIDs := es.orgIDsByGroups(ctx, cls)
	for _, cl := range cls {
		es.pub.Publish(ctx, events.Event{
			Action: events.ModbusClientCreated{
				ID:       cl.ID,
				ThingID:  cl.ThingID,
				Name:     cl.Name,
				Metadata: cl.Metadata,
			},
			GroupID: cl.GroupID,
			OrgID:   orgIDs[cl.GroupID],
		})
	}
}

// orgIDsByGroups looks up the org of each distinct group once; a failed lookup only leaves the org out.
func (es eventStore) orgIDsByGroups(ctx context.Context, cls []modbus.Client) map[string]string {
	orgIDs := make(map[string]string)
	for _, cl := range cls {
		if _, ok := orgIDs[cl.GroupID]; ok {
			continue
		}

		group, err := es.things.GetGroup(ctx, cl.GroupID)
		if err != nil {
			orgIDs[cl.GroupID] = ""
			continue
		}
		orgIDs[cl.GroupID] = group.OrgID
	}

	return orgIDs
}
