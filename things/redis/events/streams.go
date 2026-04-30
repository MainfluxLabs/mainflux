// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/things"
)

type eventStore struct {
	things.Service
	pub events.Publisher
}

func NewEventStoreMiddleware(svc things.Service, pub events.Publisher) things.Service {
	return eventStore{
		Service: svc,
		pub:     pub,
	}
}

func (es eventStore) CreateThings(ctx context.Context, token, profileID string, ths ...things.Thing) ([]things.Thing, error) {
	out, err := es.Service.CreateThings(ctx, token, profileID, ths...)
	if err != nil {
		return out, err
	}

	for _, th := range out {
		es.pub.Publish(ctx, events.ThingCreated{
			ID:        th.ID,
			GroupID:   th.GroupID,
			ProfileID: th.ProfileID,
			Name:      th.Name,
			Metadata:  th.Metadata,
		})
	}

	return out, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	if err := es.Service.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.ThingUpdated{
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

	es.pub.Publish(ctx, events.ThingGroupAndProfileUpdated{
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

		es.pub.Publish(ctx, events.ThingRemoved{ID: id})
	}

	return nil
}

func (es eventStore) CreateProfiles(ctx context.Context, token, groupID string, profiles ...things.Profile) ([]things.Profile, error) {
	prs, err := es.Service.CreateProfiles(ctx, token, groupID, profiles...)
	if err != nil {
		return prs, err
	}

	for _, pr := range prs {
		es.pub.Publish(ctx, events.ProfileCreated{
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

	es.pub.Publish(ctx, events.ProfileUpdated{
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

		es.pub.Publish(ctx, events.ProfileRemoved{ID: id})
	}

	return nil
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.GroupRemoved{ID: id})
	}

	return nil
}

func (es eventStore) RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error) {
	ids, err := es.Service.RemoveGroupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		es.pub.Publish(ctx, events.GroupRemoved{ID: id})
	}

	return ids, nil
}
