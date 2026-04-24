// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-redis/redis/v8"
)

const DefaultStreamMaxLen = 100000

type eventStore struct {
	things.Service
	client *redis.Client
	logger logger.Logger
	maxLen int64
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store. maxLen controls the approximate stream retention;
// pass 0 to use DefaultStreamMaxLen.
func NewEventStoreMiddleware(svc things.Service, client *redis.Client, maxLen int64, logger logger.Logger) things.Service {
	if maxLen <= 0 {
		maxLen = DefaultStreamMaxLen
	}

	return eventStore{
		Service: svc,
		client:  client,
		logger:  logger,
		maxLen:  maxLen,
	}
}

func (es eventStore) publish(ctx context.Context, e events.Event) {
	vals := e.Encode()
	record := &redis.XAddArgs{
		Stream:       events.ThingsStream,
		MaxLenApprox: es.maxLen,
		Values:       map[string]any(vals),
	}

	if err := es.client.XAdd(ctx, record).Err(); err != nil {
		es.logger.Warn(fmt.Sprintf("failed to publish %s event: %s", vals.Operation(), err))
	}
}

func (es eventStore) CreateThings(ctx context.Context, token, profileID string, things ...things.Thing) ([]things.Thing, error) {
	ths, err := es.Service.CreateThings(ctx, token, profileID, things...)
	if err != nil {
		return ths, err
	}

	for _, th := range ths {
		es.publish(ctx, events.ThingCreated{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Metadata:  th.Metadata,
		})
	}

	return ths, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	if err := es.Service.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	es.publish(ctx, events.ThingUpdated{
		ID:        thing.ID,
		ProfileID: thing.ProfileID,
		Name:      thing.Name,
		Metadata:  thing.Metadata,
	})

	return nil
}

func (es eventStore) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) error {
	if err := es.Service.UpdateThingGroupAndProfile(ctx, token, thing); err != nil {
		return err
	}

	es.publish(ctx, events.ThingGroupAndProfileUpdated{
		ID:        thing.ID,
		ProfileID: thing.ProfileID,
		GroupID:   thing.GroupID,
	})

	return nil
}

func (es eventStore) RemoveThings(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveThings(ctx, token, id); err != nil {
			return err
		}

		es.publish(ctx, events.ThingRemoved{ID: id})
	}

	return nil
}

func (es eventStore) CreateProfiles(ctx context.Context, token, groupID string, profiles ...things.Profile) ([]things.Profile, error) {
	prs, err := es.Service.CreateProfiles(ctx, token, groupID, profiles...)
	if err != nil {
		return prs, err
	}

	for _, pr := range prs {
		es.publish(ctx, events.ProfileCreated{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
		})
	}

	return prs, nil
}

func (es eventStore) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	if err := es.Service.UpdateProfile(ctx, token, profile); err != nil {
		return err
	}

	es.publish(ctx, events.ProfileUpdated{
		ID:       profile.ID,
		Name:     profile.Name,
		Config:   profile.Config,
		Metadata: profile.Metadata,
	})

	return nil
}

func (es eventStore) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveProfiles(ctx, token, id); err != nil {
			return err
		}

		es.publish(ctx, events.ProfileRemoved{ID: id})
	}

	return nil
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		es.publish(ctx, events.GroupRemoved{ID: id})
	}

	return nil
}

func (es eventStore) RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error) {
	ids, err := es.Service.RemoveGroupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		es.publish(ctx, events.GroupRemoved{ID: id})
	}

	return ids, nil
}
