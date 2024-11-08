// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	Viewer     = "viewer"
	Editor     = "editor"
	Admin      = "admin"
	ThingSub   = "thing"
	ChannelSub = "channel"
	GroupSub   = "group"
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
	ListThings(ctx context.Context, token string, pm PageMetadata) (ThingsPage, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (ThingsPage, error)

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
	ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error)

	// ViewChannelByThing retrieves data about channel that have
	// specified thing connected or not connected to it and belong to the user identified by
	// the provided key.
	ViewChannelByThing(ctx context.Context, token, thID string) (Channel, error)

	// RemoveChannels removes the things identified by the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveChannels(ctx context.Context, token string, ids ...string) error

	// ViewChannelProfile retrieves channel profile.
	ViewChannelProfile(ctx context.Context, chID string) (Profile, error)

	// Connect connects a list of things to a channel.
	Connect(ctx context.Context, token, chID string, thIDs []string) error

	// Disconnect disconnects a list of things from a channel.
	Disconnect(ctx context.Context, token, chID string, thIDs []string) error

	// GetConnByKey determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	GetConnByKey(ctx context.Context, key string) (Connection, error)

	// Authorize determines whether the group and its things and channels can be accessed by
	// the given user and returns error if it cannot.
	Authorize(ctx context.Context, req AuthorizeReq) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// GetProfileByThingID returns channel profile for given thing ID.
	GetProfileByThingID(ctx context.Context, thingID string) (Profile, error)

	// GetGroupIDByThingID returns a thing's group ID for given thing ID.
	GetGroupIDByThingID(ctx context.Context, thingID string) (string, error)

	// Backup retrieves all things, channels and connections for all users. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds things, channels and connections from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error

	Groups

	Policies
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64                 `json:"offset,omitempty"`
	Limit    uint64                 `json:"limit,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Order    string                 `json:"order,omitempty"`
	Dir      string                 `json:"dir,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Backup struct {
	Things      []Thing
	Channels    []Channel
	Connections []Connection
	Groups      []Group
	GroupRoles  []GroupMember
}

type AuthorizeReq struct {
	Token   string
	Object  string
	Subject string
	Action  string
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         protomfx.AuthServiceClient
	users        protomfx.UsersServiceClient
	things       ThingRepository
	channels     ChannelRepository
	groups       GroupRepository
	roles        RolesRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idProvider   uuid.IDProvider
}

// New instantiates the things service implementation.
func New(auth protomfx.AuthServiceClient, users protomfx.UsersServiceClient, things ThingRepository, channels ChannelRepository, groups GroupRepository, roles RolesRepository, ccache ChannelCache, tcache ThingCache, idp uuid.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		users:        users,
		things:       things,
		channels:     channels,
		groups:       groups,
		roles:        roles,
		channelCache: ccache,
		thingCache:   tcache,
		idProvider:   idp,
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	ths := []Thing{}
	for _, thing := range things {
		ar := AuthorizeReq{
			Token:   token,
			Object:  thing.GroupID,
			Subject: GroupSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return nil, err
		}

		th, err := ts.createThing(ctx, &thing)
		if err != nil {
			return []Thing{}, err
		}
		ths = append(ths, th)
	}

	return ths, nil
}

func (ts *thingsService) createThing(ctx context.Context, thing *Thing) (Thing, error) {
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
	ar := AuthorizeReq{
		Token:   token,
		Object:  thing.ID,
		Subject: ThingSub,
		Action:  Editor,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	ar := AuthorizeReq{
		Token:   token,
		Object:  id,
		Subject: ThingSub,
		Action:  Editor,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	return ts.things.UpdateKey(ctx, id, key)
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  id,
		Subject: ThingSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Thing{}, err
	}

	thing, err := ts.things.RetrieveByID(ctx, id)
	if err != nil {
		return Thing{}, err
	}

	return thing, nil
}

func (ts *thingsService) ListThings(ctx context.Context, token string, pm PageMetadata) (ThingsPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.things.RetrieveByAdmin(ctx, pm)
	}

	res, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ThingsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	grIDs, err := ts.roles.RetrieveGroupIDsByMember(ctx, res.GetId())
	if err != nil {
		return ThingsPage{}, err
	}

	return ts.things.RetrieveByGroupIDs(ctx, grIDs, pm)
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (ThingsPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  chID,
		Subject: ChannelSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return ThingsPage{}, err
	}

	tp, err := ts.things.RetrieveByChannel(ctx, chID, pm)
	if err != nil {
		return ThingsPage{}, err
	}

	return tp, nil
}

func (ts *thingsService) RemoveThings(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := AuthorizeReq{
			Token:   token,
			Object:  id,
			Subject: ThingSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return err
		}

		if err := ts.thingCache.Remove(ctx, id); err != nil {
			return err
		}
	}

	if err := ts.things.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	chs := []Channel{}
	for _, channel := range channels {
		ar := AuthorizeReq{
			Token:   token,
			Object:  channel.GroupID,
			Subject: GroupSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return nil, err
		}

		ch, err := ts.createChannel(ctx, &channel)
		if err != nil {
			return []Channel{}, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

func (ts *thingsService) createChannel(ctx context.Context, channel *Channel) (Channel, error) {
	if channel.ID == "" {
		chID, err := ts.idProvider.ID()
		if err != nil {
			return Channel{}, err
		}
		channel.ID = chID
	}

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
	ar := AuthorizeReq{
		Token:   token,
		Object:  channel.ID,
		Subject: ChannelSub,
		Action:  Editor,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  id,
		Subject: ChannelSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Channel{}, err
	}

	channel, err := ts.channels.RetrieveByID(ctx, id)
	if err != nil {
		return Channel{}, err
	}

	return channel, nil
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.channels.RetrieveByAdmin(ctx, pm)
	}

	res, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	grIDs, err := ts.roles.RetrieveGroupIDsByMember(ctx, res.GetId())
	if err != nil {
		return ChannelsPage{}, err
	}

	return ts.channels.RetrieveByGroupIDs(ctx, grIDs, pm)
}

func (ts *thingsService) ViewChannelByThing(ctx context.Context, token, thID string) (Channel, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  thID,
		Subject: ThingSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Channel{}, err
	}

	channel, err := ts.channels.RetrieveByThing(ctx, thID)
	if err != nil {
		return Channel{}, err
	}

	return channel, nil
}

func (ts *thingsService) RemoveChannels(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := AuthorizeReq{
			Token:   token,
			Object:  id,
			Subject: ChannelSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return err
		}

		if err := ts.channelCache.Remove(ctx, id); err != nil {
			return err
		}
	}

	return ts.channels.Remove(ctx, ids...)
}

func (ts *thingsService) ViewChannelProfile(ctx context.Context, chID string) (Profile, error) {
	channel, err := ts.channels.RetrieveByID(ctx, chID)
	if err != nil {
		return Profile{}, err
	}

	meta, err := json.Marshal(channel.Profile)
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := json.Unmarshal(meta, &profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) Connect(ctx context.Context, token, chID string, thIDs []string) error {
	ch, err := ts.channels.RetrieveByID(ctx, chID)
	if err != nil {
		return err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  chID,
		Subject: ChannelSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	for _, thID := range thIDs {
		th, err := ts.things.RetrieveByID(ctx, thID)
		if err != nil {
			return err
		}

		if th.GroupID != ch.GroupID {
			return errors.ErrAuthorization
		}

		if err := ts.channelCache.Connect(ctx, chID, thID); err != nil {
			return err
		}
	}

	return ts.channels.Connect(ctx, chID, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token, chID string, thIDs []string) error {
	ch, err := ts.channels.RetrieveByID(ctx, chID)
	if err != nil {
		return err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  chID,
		Subject: ChannelSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	for _, thID := range thIDs {
		th, err := ts.things.RetrieveByID(ctx, thID)
		if err != nil {
			return err
		}

		if th.GroupID != ch.GroupID {
			return errors.ErrAuthorization
		}

		if err := ts.channelCache.Disconnect(ctx, chID, thID); err != nil {
			return err
		}
	}

	return ts.channels.Disconnect(ctx, chID, thIDs)
}

func (ts *thingsService) GetConnByKey(ctx context.Context, thingKey string) (Connection, error) {
	conn, err := ts.channels.RetrieveConnByThingKey(ctx, thingKey)
	if err != nil {
		return Connection{}, err
	}

	if err := ts.thingCache.Save(ctx, thingKey, conn.ThingID); err != nil {
		return Connection{}, err
	}

	if err := ts.channelCache.Connect(ctx, conn.ChannelID, conn.ThingID); err != nil {
		return Connection{}, err
	}

	return Connection{ThingID: conn.ThingID, ChannelID: conn.ChannelID}, nil
}

func (ts *thingsService) Authorize(ctx context.Context, ar AuthorizeReq) error {
	var groupID string
	switch ar.Subject {
	case ThingSub:
		thing, err := ts.things.RetrieveByID(ctx, ar.Object)
		if err != nil {
			return err
		}
		groupID = thing.GroupID
	case ChannelSub:
		channel, err := ts.channels.RetrieveByID(ctx, ar.Object)
		if err != nil {
			return err
		}
		groupID = channel.GroupID
	case GroupSub:
		groupID = ar.Object
	default:
		return errors.ErrAuthorization
	}

	return ts.canAccessGroup(ctx, ar.Token, groupID, ar.Action)
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

func (ts *thingsService) GetProfileByThingID(ctx context.Context, thingID string) (Profile, error) {
	channel, err := ts.channels.RetrieveByThing(ctx, thingID)
	if err != nil {
		return Profile{}, err
	}

	meta, err := json.Marshal(channel.Profile)
	if err != nil {
		return Profile{}, err
	}

	var profile Profile
	if err := json.Unmarshal(meta, &profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) GetGroupIDByThingID(ctx context.Context, thingID string) (string, error) {
	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return "", err
	}
	return thing.GroupID, nil
}

func (ts *thingsService) Backup(ctx context.Context, token string) (Backup, error) {
	if err := ts.isAdmin(ctx, token); err != nil {
		return Backup{}, err
	}

	groups, err := ts.groups.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	groupsPolicies, err := ts.roles.RetrieveAllRolesByGroup(ctx)
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
		Things:      things,
		Channels:    channels,
		Connections: connections,
		Groups:      groups,
		GroupRoles:  groupsPolicies,
	}, nil
}

func (ts *thingsService) Restore(ctx context.Context, token string, backup Backup) error {
	if err := ts.isAdmin(ctx, token); err != nil {
		return err
	}

	for _, group := range backup.Groups {
		if _, err := ts.groups.Save(ctx, group); err != nil {
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
		if err := ts.channels.Connect(ctx, conn.ChannelID, []string{conn.ThingID}); err != nil {
			return err
		}
	}

	for _, g := range backup.GroupRoles {
		gp := GroupRoles{
			MemberID: g.MemberID,
			Role:     g.Role,
		}

		if err := ts.roles.SaveRolesByGroup(ctx, g.GroupID, gp); err != nil {
			return err
		}
	}

	return nil
}

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (ts *thingsService) ListThingsByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (ThingsPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return ThingsPage{}, err
	}

	return ts.things.RetrieveByGroupIDs(ctx, []string{groupID}, pm)
}

func (ts *thingsService) ListChannelsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ChannelsPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return ChannelsPage{}, err
	}

	return ts.groups.RetrieveChannelsByGroup(ctx, groupID, pm)
}

func (ts *thingsService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: auth.RootSub,
	}

	if _, err := ts.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (ts *thingsService) canAccessOrg(ctx context.Context, token, orgID string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Object:  orgID,
		Subject: auth.OrgSub,
		Action:  auth.Viewer,
	}

	if _, err := ts.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
