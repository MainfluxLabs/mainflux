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

	group, err := es.Service.ViewGroup(ctx, token, out[0].GroupID)
	if err != nil {
		return out, err
	}

	for _, th := range out {
		es.pub.Publish(ctx, events.Event{
			Action: events.ThingCreated{
				ID:        th.ID,
				GroupID:   th.GroupID,
				ProfileID: th.ProfileID,
				Name:      th.Name,
				Metadata:  th.Metadata,
			},
			GroupID: group.ID,
			OrgID:   group.OrgID,
		})
	}

	return out, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	groupID, err := es.Service.GetGroupIDByThing(ctx, thing.ID)
	if err != nil {
		return err
	}

	group, err := es.Service.ViewGroup(ctx, token, groupID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateThing(ctx, token, thing); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.ThingUpdated{
			ID:        thing.ID,
			ProfileID: thing.ProfileID,
			Name:      thing.Name,
			Metadata:  thing.Metadata,
		},
		GroupID: groupID,
		OrgID:   group.OrgID,
	})

	return nil
}

func (es eventStore) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) error {
	// Get Thing's current Group ID
	prevGroupID, err := es.Service.GetGroupIDByThing(ctx, thing.ID)
	if err != nil {
		return err
	}

	prevGroup, err := es.Service.ViewGroup(ctx, token, prevGroupID)
	if err != nil {
		return err
	}

	if err := es.Service.UpdateThingGroupAndProfile(ctx, token, thing); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.ThingGroupAndProfileUpdated{
			ID:        thing.ID,
			ProfileID: thing.ProfileID,
			GroupID:   thing.GroupID,
		},
		GroupID: prevGroupID,
		OrgID:   prevGroup.OrgID,
	})

	return nil
}

func (es eventStore) RemoveThings(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		groupID, err := es.Service.GetGroupIDByThing(ctx, id)
		if err != nil {
			return err
		}

		group, err := es.Service.ViewGroup(ctx, token, groupID)
		if err != nil {
			return err
		}

		if err := es.Service.RemoveThings(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action:  events.ThingRemoved{ID: id},
			GroupID: groupID,
			OrgID:   group.OrgID,
		})
	}

	return nil
}

func (es eventStore) CreateProfiles(ctx context.Context, token, groupID string, profiles ...things.Profile) ([]things.Profile, error) {
	prs, err := es.Service.CreateProfiles(ctx, token, groupID, profiles...)
	if err != nil {
		return prs, err
	}

	group, err := es.Service.ViewGroup(ctx, token, groupID)
	if err != nil {
		return prs, err
	}

	for _, pr := range prs {
		es.pub.Publish(ctx, events.Event{
			Action: events.ProfileCreated{
				ID:       pr.ID,
				GroupID:  pr.GroupID,
				Name:     pr.Name,
				Metadata: pr.Metadata,
			},
			GroupID: pr.GroupID,
			OrgID:   group.OrgID,
		})
	}

	return prs, nil
}

func (es eventStore) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	if err := es.Service.UpdateProfile(ctx, token, profile); err != nil {
		return err
	}

	group, err := es.Service.ViewGroup(ctx, token, profile.GroupID)
	if err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.ProfileUpdated{
			ID:       profile.ID,
			Name:     profile.Name,
			Config:   profile.Config,
			Metadata: profile.Metadata,
		},
		GroupID: profile.GroupID,
		OrgID:   group.OrgID,
	})

	return nil
}

func (es eventStore) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		groupID, err := es.Service.GetGroupIDByProfile(ctx, id)
		if err != nil {
			return err
		}

		group, err := es.Service.ViewGroup(ctx, token, groupID)
		if err != nil {
			return err
		}

		if err := es.Service.RemoveProfiles(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action:  events.ProfileRemoved{ID: id},
			GroupID: groupID,
			OrgID:   group.OrgID,
		})
	}

	return nil
}

func (es eventStore) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		thingIDs, err := es.Service.GetThingIDsByGroup(ctx, id)
		if err != nil {
			return err
		}

		// Obtain Org ID of Group
		group, err := es.Service.ViewGroup(ctx, token, id)
		if err != nil {
			return err
		}

		if err := es.Service.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action: events.GroupRemoved{ID: id, ThingIDs: thingIDs},
			OrgID:  group.OrgID,
		})
	}

	return nil
}

func (es eventStore) RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error) {
	groupIDs, err := es.Service.GetGroupIDsByOrgInternal(ctx, orgID)
	if err != nil {
		return nil, err
	}

	thingsByGroup := make(map[string][]string, len(groupIDs))
	for _, gid := range groupIDs {
		tids, err := es.Service.GetThingIDsByGroup(ctx, gid)
		if err != nil {
			return nil, err
		}
		thingsByGroup[gid] = tids
	}

	ids, err := es.Service.RemoveGroupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		es.pub.Publish(ctx, events.Event{
			Action: events.GroupRemoved{ID: id, ThingIDs: thingsByGroup[id]},
			OrgID:  orgID,
		})
	}

	return ids, nil
}
