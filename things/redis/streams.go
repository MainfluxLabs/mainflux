// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const streamLen = 1000

type eventStore struct {
	things.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(svc things.Service, client *redis.Client) things.Service {
	return eventStore{
		Service: svc,
		client:  client,
	}
}

func (es eventStore) CreateThings(ctx context.Context, token, profileID string, things ...things.Thing) ([]things.Thing, error) {
	ths, err := es.Service.CreateThings(ctx, token, profileID, things...)
	if err != nil {
		return ths, err
	}

	for _, th := range ths {
		event := createThingEvent{
			id:        th.ID,
			groupID:   th.GroupID,
			profileID: th.ProfileID,
			name:      th.Name,
			metadata:  th.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       events.ThingsStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return ths, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	if err := es.Service.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	event := updateThingEvent{
		id:        thing.ID,
		profileID: thing.ProfileID,
		name:      thing.Name,
		metadata:  thing.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       events.ThingsStream,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) error {
	if err := es.Service.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	event := updateThingGroupAndProfileEvent{
		id:        thing.ID,
		profileID: thing.ProfileID,
		groupID:   thing.GroupID,
	}

	record := &redis.XAddArgs{
		Stream:       events.ThingsStream,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}

	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) RemoveThings(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveThings(ctx, token, id); err != nil {
			return err
		}

		event := removeThingEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       events.ThingsStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) CreateProfiles(ctx context.Context, token, groupID string, profiles ...things.Profile) ([]things.Profile, error) {
	prs, err := es.Service.CreateProfiles(ctx, token, groupID, profiles...)
	if err != nil {
		return prs, err
	}

	for _, pr := range prs {
		event := createProfileEvent{
			id:       pr.ID,
			groupID:  pr.GroupID,
			name:     pr.Name,
			metadata: pr.Metadata,
		}
		record := &redis.XAddArgs{
			Stream:       events.ThingsStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return prs, nil
}

func (es eventStore) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	if err := es.Service.UpdateProfile(ctx, token, profile); err != nil {
		return err
	}

	event := updateProfileEvent{
		id:       profile.ID,
		name:     profile.Name,
		config:   profile.Config,
		metadata: profile.Metadata,
	}
	record := &redis.XAddArgs{
		Stream:       events.ThingsStream,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return nil
}

func (es eventStore) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveProfiles(ctx, token, id); err != nil {
			return err
		}

		event := removeProfileEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       events.ThingsStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		event := removeGroupEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       events.ThingsStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error) {
	ids, err := es.Service.RemoveGroupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		event := removeGroupEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return ids, nil
}
