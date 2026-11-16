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

	orgID, err := es.Service.GetOrgIDByGroup(ctx, out[0].GroupID)
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
			GroupID: out[0].GroupID,
			OrgID:   orgID,
		})
	}

	return out, nil
}

func (es eventStore) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	groupID, err := es.Service.GetGroupIDByThing(ctx, thing.ID)
	if err != nil {
		return err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
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
		OrgID:   orgID,
	})

	return nil
}

func (es eventStore) UpdateThingGroupAndProfile(ctx context.Context, token string, thing things.Thing) error {
	// Get Thing's current Group ID
	prevGroupID, err := es.Service.GetGroupIDByThing(ctx, thing.ID)
	if err != nil {
		return err
	}

	prevOrgID, err := es.Service.GetOrgIDByGroup(ctx, prevGroupID)
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
		OrgID:   prevOrgID,
	})

	return nil
}

func (es eventStore) RemoveThings(ctx context.Context, token string, ids ...string) error {
	grIDs := make(map[string]string, len(ids))
	orgIDs := make(map[string]string)
	for _, id := range ids {
		groupID, err := es.Service.GetGroupIDByThing(ctx, id)
		if err != nil {
			return err
		}
		grIDs[id] = groupID

		if _, ok := orgIDs[groupID]; ok {
			continue
		}

		orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
		if err != nil {
			return err
		}
		orgIDs[groupID] = orgID
	}

	if err := es.Service.RemoveThings(ctx, token, ids...); err != nil {
		return err
	}

	for _, id := range ids {
		es.pub.Publish(ctx, events.Event{
			Action:  events.ThingRemoved{ID: id},
			GroupID: grIDs[id],
			OrgID:   orgIDs[grIDs[id]],
		})
	}

	return nil
}

func (es eventStore) CreateProfiles(ctx context.Context, token, groupID string, profiles ...things.Profile) ([]things.Profile, error) {
	prs, err := es.Service.CreateProfiles(ctx, token, groupID, profiles...)
	if err != nil {
		return prs, err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
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
			OrgID:   orgID,
		})
	}

	return prs, nil
}

func (es eventStore) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	if err := es.Service.UpdateProfile(ctx, token, profile); err != nil {
		return err
	}

	groupID, err := es.Service.GetGroupIDByProfile(ctx, profile.ID)
	if err != nil {
		return err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
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
		GroupID: groupID,
		OrgID:   orgID,
	})

	return nil
}

func (es eventStore) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		groupID, err := es.Service.GetGroupIDByProfile(ctx, id)
		if err != nil {
			return err
		}

		orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
		if err != nil {
			return err
		}

		if err := es.Service.RemoveProfiles(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action:  events.ProfileRemoved{ID: id},
			GroupID: groupID,
			OrgID:   orgID,
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

		orgID, err := es.Service.GetOrgIDByGroup(ctx, id)
		if err != nil {
			return err
		}

		if err := es.Service.RemoveGroups(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action: events.GroupRemoved{ID: id, ThingIDs: thingIDs},
			OrgID:  orgID,
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

func (es eventStore) UpdateThingsMetadata(ctx context.Context, token string, ths ...things.Thing) error {
	if err := es.Service.UpdateThingsMetadata(ctx, token, ths...); err != nil {
		return err
	}

	groupIDByThing := make(map[string]string, len(ths))
	orgIDByGroup := make(map[string]string)
	for _, th := range ths {
		groupID, err := es.Service.GetGroupIDByThing(ctx, th.ID)
		if err != nil {
			return err
		}
		groupIDByThing[th.ID] = groupID

		if _, ok := orgIDByGroup[groupID]; ok {
			continue
		}

		orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
		if err != nil {
			return err
		}
		orgIDByGroup[groupID] = orgID
	}

	for _, th := range ths {
		groupID := groupIDByThing[th.ID]
		es.pub.Publish(ctx, events.Event{
			Action: events.ThingMetadataUpdated{
				ID:       th.ID,
				Metadata: th.Metadata,
			},
			GroupID: groupID,
			OrgID:   orgIDByGroup[groupID],
		})
	}

	return nil
}

func (es eventStore) UpdateExternalKey(ctx context.Context, token, key, thingID string) error {
	if err := es.Service.UpdateExternalKey(ctx, token, key, thingID); err != nil {
		return err
	}

	groupID, orgID, err := es.thingScope(ctx, thingID)
	if err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action:  events.ThingExternalKeyUpdated{ID: thingID},
		GroupID: groupID,
		OrgID:   orgID,
	})

	return nil
}

func (es eventStore) RemoveExternalKey(ctx context.Context, token, thingID string) error {
	if err := es.Service.RemoveExternalKey(ctx, token, thingID); err != nil {
		return err
	}

	groupID, orgID, err := es.thingScope(ctx, thingID)
	if err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action:  events.ThingExternalKeyRemoved{ID: thingID},
		GroupID: groupID,
		OrgID:   orgID,
	})

	return nil
}

func (es eventStore) CreateGroups(ctx context.Context, token, orgID string, groups ...things.Group) ([]things.Group, error) {
	grs, err := es.Service.CreateGroups(ctx, token, orgID, groups...)
	if err != nil {
		return grs, err
	}

	for _, gr := range grs {
		es.pub.Publish(ctx, events.Event{
			Action: events.GroupCreated{
				ID:          gr.ID,
				Name:        gr.Name,
				Description: gr.Description,
				Metadata:    gr.Metadata,
			},
			GroupID: gr.ID,
			OrgID:   orgID,
		})
	}

	return grs, nil
}

func (es eventStore) UpdateGroup(ctx context.Context, token string, group things.Group) (things.Group, error) {
	gr, err := es.Service.UpdateGroup(ctx, token, group)
	if err != nil {
		return gr, err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, group.ID)
	if err != nil {
		return things.Group{}, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.GroupUpdated{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
		},
		GroupID: group.ID,
		OrgID:   orgID,
	})

	return gr, nil
}

func (es eventStore) CreateGroupMemberships(ctx context.Context, token, redirectPath string, gms ...things.GroupMembership) error {
	if err := es.Service.CreateGroupMemberships(ctx, token, redirectPath, gms...); err != nil {
		return err
	}

	orgIDByGroup, err := es.orgIDsByGroups(ctx, gms)
	if err != nil {
		return err
	}

	for _, gm := range gms {
		es.pub.Publish(ctx, events.Event{
			Action: events.GroupMembershipCreated{
				GroupID:  gm.GroupID,
				MemberID: gm.MemberID,
				Role:     gm.Role,
			},
			GroupID: gm.GroupID,
			OrgID:   orgIDByGroup[gm.GroupID],
		})
	}

	return nil
}

func (es eventStore) UpdateGroupMemberships(ctx context.Context, token string, gms ...things.GroupMembership) error {
	if err := es.Service.UpdateGroupMemberships(ctx, token, gms...); err != nil {
		return err
	}

	orgIDByGroup, err := es.orgIDsByGroups(ctx, gms)
	if err != nil {
		return err
	}

	for _, gm := range gms {
		es.pub.Publish(ctx, events.Event{
			Action: events.GroupMembershipUpdated{
				GroupID:  gm.GroupID,
				MemberID: gm.MemberID,
				Role:     gm.Role,
			},
			GroupID: gm.GroupID,
			OrgID:   orgIDByGroup[gm.GroupID],
		})
	}

	return nil
}

func (es eventStore) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error {
	if err := es.Service.RemoveGroupMemberships(ctx, token, groupID, memberIDs...); err != nil {
		return err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
	if err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		es.pub.Publish(ctx, events.Event{
			Action: events.GroupMembershipRemoved{
				GroupID:  groupID,
				MemberID: memberID,
			},
			GroupID: groupID,
			OrgID:   orgID,
		})
	}

	return nil
}

// thingScope returns the group and org a thing belongs to.
func (es eventStore) thingScope(ctx context.Context, thingID string) (string, string, error) {
	groupID, err := es.Service.GetGroupIDByThing(ctx, thingID)
	if err != nil {
		return "", "", err
	}

	orgID, err := es.Service.GetOrgIDByGroup(ctx, groupID)
	if err != nil {
		return "", "", err
	}

	return groupID, orgID, nil
}

// orgIDsByGroups returns the org ID of each distinct group in the memberships.
func (es eventStore) orgIDsByGroups(ctx context.Context, gms []things.GroupMembership) (map[string]string, error) {
	orgIDByGroup := make(map[string]string)
	for _, gm := range gms {
		if _, ok := orgIDByGroup[gm.GroupID]; ok {
			continue
		}

		orgID, err := es.Service.GetOrgIDByGroup(ctx, gm.GroupID)
		if err != nil {
			return nil, err
		}
		orgIDByGroup[gm.GroupID] = orgID
	}

	return orgIDByGroup, nil
}
