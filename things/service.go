// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"

	"github.com/MainfluxLabs/mainflux"
)

const rootSubject = "root"

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

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(ctx context.Context, token, id string) error

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

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected or not connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(ctx context.Context, token, id string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, token string, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected
	// things.
	Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error

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

	// CreateGroup creates new  group.
	CreateGroup(ctx context.Context, token string, g Group) (Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token string, admin bool, pm PageMetadata) (GroupPage, error)

	// ListGroupsByIDs retrieves groups by their IDs.
	ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error)

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token string, groupID string, pm PageMetadata) (MemberPage, error)

	// ViewMembership retrieves group that member belongs to.
	ViewMembership(ctx context.Context, token, memberID string) (Group, error)

	// RemoveGroup removes the group identified with the provided ID.
	RemoveGroup(ctx context.Context, token, id string) error

	// Assign adds a member with memberID into the group identified by groupID.
	Assign(ctx context.Context, token, groupID string, memberIDs ...string) error

	// Unassign removes member with memberID from group identified by groupID.
	Unassign(ctx context.Context, token, groupID string, memberIDs ...string) error
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
}

type Backup struct {
	Things         []Thing
	Channels       []Channel
	Connections    []Connection
	Groups         []Group
	GroupRelations []GroupRelation
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

// createThing saves the Thing and adds identity as an owner(Read, Write, Delete policies) of the Thing.
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

	if err := ts.authorize(ctx, rootSubject, token); err == nil {
		return thing, nil
	}

	if thing.Owner == res.GetId() {
		return thing, nil
	}

	groupID, err := ts.groups.RetrieveMembership(ctx, id)
	if err != nil {
		return Thing{}, errors.ErrAuthorization
	}

	if _, err = ts.auth.CanAccessGroup(ctx, &mainflux.AccessGroupReq{Token: token, GroupID: groupID}); err == nil {
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
		if err := ts.authorize(ctx, rootSubject, token); err == nil {
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

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	if _, err = ts.things.RetrieveByID(ctx, id); err != nil {
		return err
	}

	if err := ts.thingCache.Remove(ctx, id); err != nil {
		return err
	}

	return ts.things.Remove(ctx, res.GetId(), id)
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

	if err := ts.authorize(ctx, rootSubject, token); err == nil {
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
		if err := ts.authorize(ctx, rootSubject, token); err == nil {
			return ts.channels.RetrieveByAdmin(ctx, pm)
		}
	}

	return ts.channels.RetrieveByOwner(ctx, res.GetId(), pm)
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	thing, err := ts.things.RetrieveByID(ctx, thID)
	if err != nil {
		return ChannelsPage{}, err
	}

	if err := ts.authorize(ctx, rootSubject, token); err == nil {
		return ts.channels.RetrieveByThing(ctx, res.GetId(), thID, pm)
	}

	if thing.Owner == res.GetId() {
		return ts.channels.RetrieveByThing(ctx, res.GetId(), thID, pm)
	}

	groupID, err := ts.groups.RetrieveMembership(ctx, thID)
	if err != nil {
		return ChannelsPage{}, err
	}

	if _, err = ts.auth.CanAccessGroup(ctx, &mainflux.AccessGroupReq{Token: token, GroupID: groupID}); err == nil {
		return ts.channels.RetrieveConns(ctx, thID, pm)
	}

	return ChannelsPage{}, errors.ErrAuthorization
}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	if _, err = ts.channels.RetrieveByID(ctx, id); err != nil {
		return err
	}

	if err := ts.channelCache.Remove(ctx, id); err != nil {
		return err
	}

	return ts.channels.Remove(ctx, res.GetId(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, thID := range thIDs {
		if err := ts.isThingOwner(ctx, res.GetId(), thID); err != nil {
			return err
		}
	}

	for _, chID := range chIDs {
		if err := ts.IsChannelOwner(ctx, res.GetId(), chID); err != nil {
			return err
		}
	}

	return ts.channels.Connect(ctx, res.GetId(), chIDs, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	for _, chID := range chIDs {
		for _, thID := range thIDs {
			if err := ts.channelCache.Disconnect(ctx, chID, thID); err != nil {
				return err
			}
		}
	}

	return ts.channels.Disconnect(ctx, res.GetId(), chIDs, thIDs)
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
	_, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Backup{}, err
	}

	if err := ts.authorize(ctx, rootSubject, token); err != nil {
		return Backup{}, err
	}

	groups, err := ts.groups.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	groupRelations, err := ts.groups.RetrieveAllGroupRelations(ctx)
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
		Things:         things,
		Channels:       channels,
		Connections:    connections,
		Groups:         groups,
		GroupRelations: groupRelations,
	}, nil
}

func (ts *thingsService) Restore(ctx context.Context, token string, backup Backup) error {
	_, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, rootSubject, token); err != nil {
		return err
	}

	for _, group := range backup.Groups {
		_, err = ts.groups.Save(ctx, group)
		if err != nil {
			return err
		}
	}

	for _, grRel := range backup.GroupRelations {
		err = ts.groups.AssignMember(ctx, grRel.GroupID, grRel.MemberID)
		if err != nil {
			return err
		}
	}

	_, err = ts.things.Save(ctx, backup.Things...)
	if err != nil {
		return err
	}

	_, err = ts.channels.Save(ctx, backup.Channels...)
	if err != nil {
		return err
	}

	for _, conn := range backup.Connections {
		err = ts.channels.Connect(ctx, conn.ThingOwner, []string{conn.ChannelID}, []string{conn.ThingID})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) CreateGroup(ctx context.Context, token string, group Group) (Group, error) {
	user, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Group{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	id, err := ts.idProvider.ID()
	if err != nil {
		return Group{}, err
	}

	timestamp := getTimestmap()
	group.UpdatedAt = timestamp
	group.CreatedAt = timestamp

	group.ID = id
	group.OwnerID = user.Id

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
		if err := ts.authorize(ctx, rootSubject, token); err == nil {
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

func (ts *thingsService) RemoveGroup(ctx context.Context, token, id string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return err
	}

	if _, err := ts.groups.RetrieveByID(ctx, id); err != nil {
		return err
	}

	return ts.groups.Remove(ctx, id)
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Group{}, err
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

	_, err = ts.auth.CanAccessGroup(ctx, &mainflux.AccessGroupReq{Token: token, GroupID: id})
	if user.GetId() != gr.OwnerID && err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) Assign(ctx context.Context, token string, groupID string, memberIDs ...string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return err
	}

	if err := ts.groups.AssignMember(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) Unassign(ctx context.Context, token string, groupID string, memberIDs ...string) error {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return err
	}

	return ts.groups.UnassignMember(ctx, groupID, memberIDs...)
}

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (ts *thingsService) ListMembers(ctx context.Context, token string, groupID string, pm PageMetadata) (MemberPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return MemberPage{}, err
	}

	mp, err := ts.groups.RetrieveMembers(ctx, groupID, pm)
	if err != nil {
		return MemberPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}

	return mp, nil
}

func (ts *thingsService) ViewMembership(ctx context.Context, token string, memberID string) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Group{}, err
	}
	groupID, err := ts.groups.RetrieveMembership(ctx, memberID)
	if err != nil {
		return Group{}, err
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
