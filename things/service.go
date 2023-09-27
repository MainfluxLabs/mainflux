// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings adds things to the user identified by the provided key.
	CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(ctx context.Context, token string, thing Thing) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(ctx context.Context, token, id, key string) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(ctx context.Context, token, id string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(ctx context.Context, token string, admin bool, pm PageMetadata) (Page, error)

	// ListThingsByIDs retrieves data about subset of things that are identified
	ListThingsByIDs(ctx context.Context, ids []string) (Page, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error)

	// RemoveThings removes the things identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveThings(ctx context.Context, token string, id ...string) error

	// CreateChannels adds channels to the user identified by the provided key.
	CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(ctx context.Context, token string, channel Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(ctx context.Context, token, id string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(ctx context.Context, token string, admin bool, pm PageMetadata) (ChannelsPage, error)

	// ViewChannelByThing retrieves data about channel that have
	// specified thing connected or not connected to it and belong to the user identified by
	// the provided key.
	ViewChannelByThing(ctx context.Context, token, thID string) (Channel, error)

	// RemoveChannels removes the things identified by the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveChannels(ctx context.Context, token string, ids ...string) error

	// Connect connects a list of things to a channel.
	Connect(ctx context.Context, token, chID string, thIDs []string) error

	// Disconnect disconnects a list of things from a channel.
	Disconnect(ctx context.Context, token, chID string, thIDs []string) error

	// CanAccessByKey determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccessByKey(ctx context.Context, chanID, key string) (string, error)

	// CanAccessByID determines whether the channel can be accessed by
	// the given thing and returns error if it cannot.
	CanAccessByID(ctx context.Context, chanID, thingID string) error

	// IsChannelOwner determines whether the channel can be accessed by
	// the given user and returns error if it cannot.
	IsChannelOwner(ctx context.Context, owner, chanID string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// Backup retrieves all things, channels and connections for all users. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds things, channels and connections from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error

	// CreateGroups adds groups to the user identified by the provided key.
	CreateGroups(ctx context.Context, token string, groups ...Group) ([]Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token string, admin bool, pm PageMetadata) (GroupPage, error)

	// ListGroupsByIDs retrieves groups by their IDs.
	ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error)

	// ListGroupThings retrieves page of things that are assigned to a group identified by groupID.
	ListGroupThings(ctx context.Context, token string, groupID string, pm PageMetadata) (GroupThingsPage, error)

	// ListGroupThingsByChannel retrieves page of disconnected things by channel that are assigned to a group same as channel.
	ListGroupThingsByChannel(ctx context.Context, token, grID, chID string, pm PageMetadata) (GroupThingsPage, error)

	// ViewThingMembership retrieves group that thing belongs to.
	ViewThingMembership(ctx context.Context, token, thingID string) (Group, error)

	// RemoveGroups removes the groups identified with the provided IDs.
	RemoveGroups(ctx context.Context, token string, ids ...string) error

	// AssignThing adds a thing with thingID into the group identified by groupID.
	AssignThing(ctx context.Context, token, groupID string, thingIDs ...string) error

	// UnassignThing removes thing with thingID from group identified by groupID.
	UnassignThing(ctx context.Context, token, groupID string, thingIDs ...string) error

	// ListGroupChannels retrieves page of channels that are assigned to a group identified by groupID.
	ListGroupChannels(ctx context.Context, token, groupID string, pm PageMetadata) (GroupChannelsPage, error)

	// ViewChannelMembership retrieves group that channel belongs to.
	ViewChannelMembership(ctx context.Context, token, channelID string) (Group, error)

	// AssignChannel adds channel to the group identified by groupID.
	AssignChannel(ctx context.Context, token string, groupID string, channelIDs ...string) error

	// UnassignChannel removes channels from the group identified by groupID.
	UnassignChannel(ctx context.Context, token string, groupID string, channelIDs ...string) error
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total        uint64
	Offset       uint64                 `json:"offset,omitempty"`
	Limit        uint64                 `json:"limit,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Order        string                 `json:"order,omitempty"`
	Dir          string                 `json:"dir,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Disconnected bool                   // Used for connected or disconnected lists
	Unassigned   bool                   // Used for assigned or unassigned lists
}

type Backup struct {
	Things              []Thing
	Channels            []Channel
	Connections         []Connection
	Groups              []Group
	GroupThingRelations []GroupThingRelation
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         mainflux.AuthServiceClient
	things       ThingRepository
	channels     ChannelRepository
	groups       GroupRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idProvider   mainflux.IDProvider
}

// New instantiates the things service implementation.
func New(auth mainflux.AuthServiceClient, things ThingRepository, channels ChannelRepository, groups GroupRepository, ccache ChannelCache, tcache ThingCache, idp mainflux.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		things:       things,
		channels:     channels,
		groups:       groups,
		channelCache: ccache,
		thingCache:   tcache,
		idProvider:   idp,
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Thing{}, err
	}

	ths := []Thing{}
	for _, thing := range things {
		th, err := ts.createThing(ctx, &thing, res)

		if err != nil {
			return []Thing{}, err
		}
		ths = append(ths, th)
	}

	return ths, nil
}

func (ts *thingsService) createThing(ctx context.Context, thing *Thing, identity *mainflux.UserIdentity) (Thing, error) {
	thing.Owner = identity.GetId()

	if thing.ID == "" {
		id, err := ts.idProvider.ID()
		if err != nil {
			return Thing{}, err
		}
		thing.ID = id
	}

	if thing.Key == "" {
		key, err := ts.idProvider.ID()

		if err != nil {
			return Thing{}, err
		}
		thing.Key = key
	}

	ths, err := ts.things.Save(ctx, *thing)
	if err != nil {
		return Thing{}, err
	}
	if len(ths) == 0 {
		return Thing{}, errors.ErrCreateEntity
	}
	return ths[0], nil
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.isThingOwner(ctx, res.GetId(), thing.ID); err != nil {
		return err
	}

	thing.Owner = res.GetId()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	owner := res.GetId()

	return ts.things.UpdateKey(ctx, owner, id, key)
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	thing, err := ts.things.RetrieveByID(ctx, id)
	if err != nil {
		return Thing{}, err
	}

	if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
		return thing, nil
	}

	if thing.Owner == res.GetId() {
		return thing, nil
	}

	groupID, err := ts.groups.RetrieveThingMembership(ctx, id)
	if err != nil {
		return Thing{}, errors.ErrAuthorization
	}

	if _, err = ts.auth.Authorize(ctx, &mainflux.AuthorizeReq{Token: token, Subject: auth.GroupSubject, Object: groupID, Action: auth.ReadAction}); err == nil {
		return thing, nil
	}

	return Thing{}, errors.ErrAuthorization
}

func (ts *thingsService) ListThings(ctx context.Context, token string, admin bool, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	if admin {
		if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
			return ts.things.RetrieveByAdmin(ctx, pm)
		}
	}

	return ts.things.RetrieveByOwner(ctx, res.GetId(), pm)
}

func (ts *thingsService) ListThingsByIDs(ctx context.Context, ids []string) (Page, error) {
	things, err := ts.things.RetrieveByIDs(ctx, ids, PageMetadata{})
	if err != nil {
		return Page{}, err
	}
	return things, nil
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return ts.things.RetrieveByChannel(ctx, res.GetId(), chID, pm)
}

func (ts *thingsService) RemoveThings(ctx context.Context, token string, ids ...string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, id := range ids {
		if err := ts.thingCache.Remove(ctx, id); err != nil {
			return err
		}
	}

	if err := ts.things.Remove(ctx, res.GetId(), ids...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Channel{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	chs := []Channel{}
	for _, channel := range channels {
		ch, err := ts.createChannel(ctx, &channel, res)
		if err != nil {
			return []Channel{}, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

func (ts *thingsService) createChannel(ctx context.Context, channel *Channel, identity *mainflux.UserIdentity) (Channel, error) {
	if channel.ID == "" {
		chID, err := ts.idProvider.ID()
		if err != nil {
			return Channel{}, err
		}
		channel.ID = chID
	}
	channel.Owner = identity.GetId()

	chs, err := ts.channels.Save(ctx, *channel)
	if err != nil {
		return Channel{}, err
	}
	if len(chs) == 0 {
		return Channel{}, errors.ErrCreateEntity
	}

	return chs[0], nil
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	channel.Owner = res.GetId()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	channel, err := ts.channels.RetrieveByID(ctx, id)
	if err != nil {
		return Channel{}, err
	}

	if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
		return channel, nil
	}

	if channel.Owner != res.GetId() {
		return Channel{}, errors.ErrAuthorization
	}

	return channel, nil
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, admin bool, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	if admin {
		if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
			return ts.channels.RetrieveByAdmin(ctx, pm)
		}
	}

	return ts.channels.RetrieveByOwner(ctx, res.GetId(), pm)
}

func (ts *thingsService) ViewChannelByThing(ctx context.Context, token, thID string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	thing, err := ts.things.RetrieveByID(ctx, thID)
	if err != nil {
		return Channel{}, err
	}

	if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
		return ts.channels.RetrieveByThing(ctx, res.GetId(), thID)
	}

	if thing.Owner == res.GetId() {
		return ts.channels.RetrieveByThing(ctx, res.GetId(), thID)
	}

	groupID, err := ts.groups.RetrieveThingMembership(ctx, thID)
	if err != nil {
		return Channel{}, err
	}

	if _, err = ts.auth.Authorize(ctx, &mainflux.AuthorizeReq{Token: token, Subject: auth.GroupSubject, Object: groupID, Action: auth.ReadAction}); err == nil {
		return ts.channels.RetrieveByThing(ctx, res.GetId(), thID)
	}

	return Channel{}, errors.ErrAuthorization
}

func (ts *thingsService) RemoveChannels(ctx context.Context, token string, ids ...string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, id := range ids {
		if err := ts.channelCache.Remove(ctx, id); err != nil {
			return err
		}
	}

	return ts.channels.Remove(ctx, res.GetId(), ids...)
}

func (ts *thingsService) Connect(ctx context.Context, token, chID string, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, thID := range thIDs {
		if err := ts.isThingOwner(ctx, res.GetId(), thID); err != nil {
			return err
		}
	}

	if err := ts.IsChannelOwner(ctx, res.GetId(), chID); err != nil {
		return err
	}

	cgrID, err := ts.groups.RetrieveChannelMembership(ctx, chID)
	if err != nil {
		return err
	}

	if cgrID == "" {
		return errors.ErrAuthorization
	}

	for _, thID := range thIDs {
		tgrID, err := ts.groups.RetrieveThingMembership(ctx, thID)
		if err != nil {
			return err
		}

		if tgrID != cgrID {
			return errors.ErrAuthorization
		}
	}

	return ts.channels.Connect(ctx, res.GetId(), chID, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token, chID string, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, thID := range thIDs {
		if err := ts.channelCache.Disconnect(ctx, chID, thID); err != nil {
			return err
		}
	}

	return ts.channels.Disconnect(ctx, res.GetId(), chID, thIDs)
}

func (ts *thingsService) CanAccessByKey(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.hasThing(ctx, chanID, thingKey)
	if err == nil {
		return thingID, nil
	}

	thingID, err = ts.channels.HasThing(ctx, chanID, thingKey)
	if err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, thingKey, thingID); err != nil {
		return "", err
	}
	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return "", err
	}
	return thingID, nil
}

func (ts *thingsService) CanAccessByID(ctx context.Context, chanID, thingID string) error {
	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); connected {
		return nil
	}

	if err := ts.channels.HasThingByID(ctx, chanID, thingID); err != nil {
		return err
	}

	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return err
	}
	return nil
}

func (ts *thingsService) IsChannelOwner(ctx context.Context, owner, chanID string) error {
	ch, err := ts.channels.RetrieveByID(ctx, chanID)
	if err != nil {
		return err
	}

	if ch.Owner != owner {
		return errors.ErrAuthorization
	}

	return nil
}

func (ts *thingsService) Identify(ctx context.Context, key string) (string, error) {
	id, err := ts.thingCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	id, err = ts.things.RetrieveByKey(ctx, key)
	if err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, key, id); err != nil {
		return "", err
	}
	return id, nil
}

func (ts *thingsService) hasThing(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.thingCache.ID(ctx, thingKey)
	if err != nil {
		return "", err
	}

	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); !connected {
		return "", errors.ErrAuthorization
	}
	return thingID, nil
}

func (ts *thingsService) Backup(ctx context.Context, token string) (Backup, error) {
	if err := ts.authorize(ctx, auth.RootSubject, token); err != nil {
		return Backup{}, err
	}

	groups, err := ts.groups.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	groupThingRelations, err := ts.groups.RetrieveAllThingRelations(ctx)
	if err != nil {
		return Backup{}, err
	}

	things, err := ts.things.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	channels, err := ts.channels.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	connections, err := ts.channels.RetrieveAllConnections(ctx)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		Things:              things,
		Channels:            channels,
		Connections:         connections,
		Groups:              groups,
		GroupThingRelations: groupThingRelations,
	}, nil
}

func (ts *thingsService) Restore(ctx context.Context, token string, backup Backup) error {
	if err := ts.authorize(ctx, auth.RootSubject, token); err != nil {
		return err
	}

	for _, group := range backup.Groups {
		if _, err := ts.groups.Save(ctx, group); err != nil {
			return err
		}
	}

	for _, gtr := range backup.GroupThingRelations {
		if err := ts.groups.AssignThing(ctx, gtr.GroupID, gtr.ThingID); err != nil {
			return err
		}
	}

	if _, err := ts.things.Save(ctx, backup.Things...); err != nil {
		return err
	}

	if _, err := ts.channels.Save(ctx, backup.Channels...); err != nil {
		return err
	}

	for _, conn := range backup.Connections {
		if err := ts.channels.Connect(ctx, conn.ThingOwner, conn.ChannelID, []string{conn.ThingID}); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) CreateGroups(ctx context.Context, token string, groups ...Group) ([]Group, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Group{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	owner := user.GetId()
	timestamp := getTimestmap()

	grs := []Group{}
	for _, group := range groups {
		group.OwnerID = owner
		group.CreatedAt = timestamp
		group.UpdatedAt = timestamp

		gr, err := ts.createGroup(ctx, group)
		if err != nil {
			return []Group{}, err
		}

		grs = append(grs, gr)
	}

	return grs, nil
}

func (ts *thingsService) createGroup(ctx context.Context, group Group) (Group, error) {
	id, err := ts.idProvider.ID()
	if err != nil {
		return Group{}, err
	}
	group.ID = id

	group, err = ts.groups.Save(ctx, group)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) ListGroups(ctx context.Context, token string, admin bool, pm PageMetadata) (GroupPage, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return GroupPage{}, err
	}

	if admin {
		if err := ts.authorize(ctx, auth.RootSubject, token); err == nil {
			return ts.groups.RetrieveByAdmin(ctx, pm)
		}
	}

	return ts.groups.RetrieveByOwner(ctx, user.GetId(), pm)
}

func (ts *thingsService) ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error) {
	page, err := ts.groups.RetrieveByIDs(ctx, ids)
	if err != nil {
		return []Group{}, err
	}

	return page.Groups, nil
}

func (ts *thingsService) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	for _, id := range ids {
		gr, err := ts.groups.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}

		if gr.OwnerID != user.GetId() {
			return errors.ErrAuthorization
		}

		cp, err := ts.groups.RetrieveGroupChannels(ctx, user.GetId(), id, PageMetadata{})
		if err != nil {
			return err
		}

		for _, ch := range cp.Channels {
			tp, err := ts.things.RetrieveByChannel(ctx, user.GetId(), ch.ID, PageMetadata{})
			if err != nil {
				return err
			}

			var thingIDs []string
			for _, th := range tp.Things {
				thingIDs = append(thingIDs, th.ID)
			}

			if err := ts.channels.Disconnect(ctx, user.GetId(), ch.ID, thingIDs); err != nil {
				return err
			}
		}
	}

	return ts.groups.Remove(ctx, ids...)
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return Group{}, err
	}

	if gr.OwnerID != user.GetId() {
		return Group{}, errors.ErrAuthorization
	}

	group.UpdatedAt = getTimestmap()

	return ts.groups.Update(ctx, group)
}

func (ts *thingsService) ViewGroup(ctx context.Context, token, id string) (Group, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, id)
	if err != nil {
		return Group{}, errors.ErrNotFound
	}

	_, err = ts.auth.Authorize(ctx, &mainflux.AuthorizeReq{Token: token, Subject: auth.GroupSubject, Object: id, Action: auth.ReadAction})
	if user.GetId() != gr.OwnerID && err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) AssignThing(ctx context.Context, token string, groupID string, thingIDs ...string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return err
	}

	for _, thingID := range thingIDs {
		thing, err := ts.things.RetrieveByID(ctx, thingID)
		if err != nil {
			return err
		}

		if thing.ID == "" {
			return errors.ErrNotFound
		}
	}

	if err := ts.groups.AssignThing(ctx, groupID, thingIDs...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) AssignChannel(ctx context.Context, token string, groupID string, channelIDs ...string) error {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	if user.GetId() != group.OwnerID {
		return errors.ErrAuthorization
	}

	for _, channelID := range channelIDs {
		ch, err := ts.channels.RetrieveByID(ctx, channelID)
		if err != nil {
			return err
		}

		if ch.ID == "" {
			return errors.ErrNotFound
		}
	}

	if err := ts.groups.AssignChannel(ctx, groupID, channelIDs...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) UnassignChannel(ctx context.Context, token string, groupID string, channelIDs ...string) error {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	if user.GetId() != group.OwnerID {
		return errors.ErrAuthorization
	}

	for _, chID := range channelIDs {
		tp, err := ts.things.RetrieveByChannel(ctx, user.GetId(), chID, PageMetadata{})
		if err != nil {
			return err
		}

		var thingIDs []string
		for _, th := range tp.Things {
			thingIDs = append(thingIDs, th.ID)
		}

		if err := ts.channels.Disconnect(ctx, user.GetId(), chID, thingIDs); err != nil {
			return err
		}
	}

	if err := ts.groups.UnassignChannel(ctx, groupID, channelIDs...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) UnassignThing(ctx context.Context, token string, groupID string, thingIDs ...string) error {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	if user.GetId() != group.OwnerID {
		return errors.ErrAuthorization
	}

	for _, thingID := range thingIDs {
		ch, err := ts.channels.RetrieveByThing(ctx, user.GetId(), thingID)
		if err != nil {
			return err
		}

		if ch.ID != "" {
			if err := ts.channels.Disconnect(ctx, user.GetId(), ch.ID, []string{thingID}); err != nil {
				return err
			}
		}
	}

	return ts.groups.UnassignThing(ctx, groupID, thingIDs...)
}

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (ts *thingsService) ListGroupThings(ctx context.Context, token string, groupID string, pm PageMetadata) (GroupThingsPage, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return GroupThingsPage{}, err
	}

	gthp, err := ts.groups.RetrieveGroupThings(ctx, user.GetId(), groupID, pm)
	if err != nil {
		return GroupThingsPage{}, errors.Wrap(ErrRetrieveGroupThings, err)
	}

	return gthp, nil
}

func (ts *thingsService) ListGroupThingsByChannel(ctx context.Context, token, grID, chID string, pm PageMetadata) (GroupThingsPage, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return GroupThingsPage{}, err
	}

	group, err := ts.groups.RetrieveByID(ctx, grID)
	if err != nil {
		return GroupThingsPage{}, err
	}

	_, err = ts.auth.Authorize(ctx, &mainflux.AuthorizeReq{Token: token, Subject: auth.GroupSubject, Object: grID, Action: auth.ReadAction})
	if user.GetId() != group.OwnerID && err != nil {
		return GroupThingsPage{}, err
	}

	return ts.groups.RetrieveGroupThingsByChannel(ctx, grID, chID, pm)
}

func (ts *thingsService) ListGroupChannels(ctx context.Context, token, groupID string, pm PageMetadata) (GroupChannelsPage, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return GroupChannelsPage{}, err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return GroupChannelsPage{}, err
	}

	if user.GetId() != group.OwnerID {
		return GroupChannelsPage{}, errors.ErrAuthorization
	}

	gchp, err := ts.groups.RetrieveGroupChannels(ctx, user.GetId(), groupID, pm)
	if err != nil {
		return GroupChannelsPage{}, err
	}

	return gchp, nil
}

func (ts *thingsService) ViewThingMembership(ctx context.Context, token string, thingID string) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Group{}, err
	}

	groupID, err := ts.groups.RetrieveThingMembership(ctx, thingID)
	if err != nil {
		return Group{}, err
	}

	if groupID == "" {
		return Group{}, nil
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) ViewChannelMembership(ctx context.Context, token string, channelID string) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Group{}, err
	}

	groupID, err := ts.groups.RetrieveChannelMembership(ctx, channelID)
	if err != nil {
		return Group{}, err
	}

	if groupID == "" {
		return Group{}, errors.Wrap(errors.ErrNotFound, err)
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) authorize(ctx context.Context, subject, token string) error {
	req := &mainflux.AuthorizeReq{
		Token:   token,
		Subject: subject,
	}

	if _, err := ts.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (ts *thingsService) isThingOwner(ctx context.Context, owner string, thingID string) error {
	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return err
	}

	if thing.Owner != owner {
		return errors.ErrNotFound
	}

	return nil
}
