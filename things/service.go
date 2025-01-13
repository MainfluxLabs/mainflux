// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
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
	Owner      = "owner"
	ThingSub   = "thing"
	ProfileSub = "profile"
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

	// ListThingsByProfile retrieves data about subset of things that are
	// connected or not connected to specified profile and belong to the user identified by
	// the provided key.
	ListThingsByProfile(ctx context.Context, token, prID string, pm PageMetadata) (ThingsPage, error)

	// RemoveThings removes the things identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveThings(ctx context.Context, token string, id ...string) error

	// CreateProfiles adds profiles to the user identified by the provided key.
	CreateProfiles(ctx context.Context, token string, profiles ...Profile) ([]Profile, error)

	// UpdateProfile updates the profile identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateProfile(ctx context.Context, token string, profile Profile) error

	// ViewProfile retrieves data about the profile identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewProfile(ctx context.Context, token, id string) (Profile, error)

	// ListProfiles retrieves data about subset of profiles that belongs to the
	// user identified by the provided key.
	ListProfiles(ctx context.Context, token string, pm PageMetadata) (ProfilesPage, error)

	// ViewProfileByThing retrieves data about profile that have
	// specified thing connected or not connected to it and belong to the user identified by
	// the provided key.
	ViewProfileByThing(ctx context.Context, token, thID string) (Profile, error)

	// RemoveProfiles removes the things identified by the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveProfiles(ctx context.Context, token string, ids ...string) error

	// GetPubConfByKey determines whether the profile can be accessed using the
	// provided key and returns thing's id if access is allowed.
	GetPubConfByKey(ctx context.Context, key string) (PubConfInfo, error)

	// GetConfigByThingID returns profile config for given thing ID.
	GetConfigByThingID(ctx context.Context, thingID string) (map[string]interface{}, error)

	// Authorize determines whether the group and its things and profiles can be accessed by
	// the given user and returns error if it cannot.
	Authorize(ctx context.Context, req AuthorizeReq) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// GetGroupIDByThingID returns a thing's group ID for given thing ID.
	GetGroupIDByThingID(ctx context.Context, thingID string) (string, error)

	// Backup retrieves all things, profiles, groups, and groups roles for all users. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds things, profiles, groups, and groups roles from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error

	Groups

	Roles
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
	Things     []Thing
	Profiles   []Profile
	Groups     []Group
	GroupRoles []GroupMember
}

type AuthorizeReq struct {
	Token   string
	Object  string
	Subject string
	Action  string
}

type PubConfInfo struct {
	PublisherID   string
	ProfileConfig map[string]interface{}
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         protomfx.AuthServiceClient
	users        protomfx.UsersServiceClient
	things       ThingRepository
	profiles     ProfileRepository
	groups       GroupRepository
	roles        RolesRepository
	profileCache ProfileCache
	thingCache   ThingCache
	groupCache   GroupCache
	idProvider   uuid.IDProvider
}

// New instantiates the things service implementation.
func New(auth protomfx.AuthServiceClient, users protomfx.UsersServiceClient, things ThingRepository, profiles ProfileRepository, groups GroupRepository, roles RolesRepository, pcache ProfileCache, tcache ThingCache, gcache GroupCache, idp uuid.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		users:        users,
		things:       things,
		profiles:     profiles,
		groups:       groups,
		roles:        roles,
		profileCache: pcache,
		thingCache:   tcache,
		groupCache:   gcache,
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

		prGrID, err := ts.profileCache.GroupID(ctx, thing.ProfileID)
		if err != nil {
			profile, err := ts.profiles.RetrieveByID(ctx, thing.ProfileID)
			if err != nil {
				return []Thing{}, err
			}
			prGrID = profile.GroupID

			if err := ts.profileCache.SaveGroupID(ctx, profile.ID, profile.GroupID); err != nil {
				return []Thing{}, err
			}
		}

		if prGrID != thing.GroupID {
			return nil, errors.ErrAuthorization
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

	if err := ts.thingCache.SaveGroupID(ctx, thing.ID, thing.GroupID); err != nil {
		return Thing{}, err
	}

	return ths[0], nil
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	thGroup, err := ts.thingCache.GroupID(ctx, thing.ID)
	if err != nil {
		th, err := ts.things.RetrieveByID(ctx, thing.ID)
		if err != nil {
			return err
		}
		thGroup = th.GroupID

		if err := ts.thingCache.SaveGroupID(ctx, th.ID, th.GroupID); err != nil {
			return err
		}
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  thGroup,
		Subject: GroupSub,
		Action:  Editor,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	prGrID, err := ts.profileCache.GroupID(ctx, thing.ProfileID)
	if err != nil {
		profile, err := ts.profiles.RetrieveByID(ctx, thing.ProfileID)
		if err != nil {
			return err
		}
		prGrID = profile.GroupID

		if err := ts.profileCache.SaveGroupID(ctx, profile.ID, profile.GroupID); err != nil {
			return err
		}
	}

	if prGrID != thGroup {
		return errors.ErrAuthorization
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
	thing, err := ts.things.RetrieveByID(ctx, id)
	if err != nil {
		return Thing{}, err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  thing.GroupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
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

	grIDs, err := ts.groupCache.GroupIDsByMember(ctx, res.GetId())
	if err != nil {
		grIDs, err = ts.roles.RetrieveGroupIDsByMember(ctx, res.GetId())
		if err != nil {
			return ThingsPage{}, err
		}
	}

	return ts.things.RetrieveByGroupIDs(ctx, grIDs, pm)
}

func (ts *thingsService) ListThingsByProfile(ctx context.Context, token, prID string, pm PageMetadata) (ThingsPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  prID,
		Subject: ProfileSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return ThingsPage{}, err
	}

	tp, err := ts.things.RetrieveByProfile(ctx, prID, pm)
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

		if err := ts.thingCache.RemoveGroupID(ctx, id); err != nil {
			return err
		}
	}

	if err := ts.things.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) CreateProfiles(ctx context.Context, token string, profiles ...Profile) ([]Profile, error) {
	prs := []Profile{}
	for _, profile := range profiles {
		ar := AuthorizeReq{
			Token:   token,
			Object:  profile.GroupID,
			Subject: GroupSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return nil, err
		}

		pr, err := ts.createProfile(ctx, &profile)
		if err != nil {
			return []Profile{}, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func (ts *thingsService) createProfile(ctx context.Context, profile *Profile) (Profile, error) {
	if profile.ID == "" {
		prID, err := ts.idProvider.ID()
		if err != nil {
			return Profile{}, err
		}
		profile.ID = prID
	}

	prs, err := ts.profiles.Save(ctx, *profile)
	if err != nil {
		return Profile{}, err
	}
	if len(prs) == 0 {
		return Profile{}, errors.ErrCreateEntity
	}

	if err := ts.profileCache.SaveGroupID(ctx, profile.ID, profile.GroupID); err != nil {
		return Profile{}, err
	}

	return prs[0], nil
}

func (ts *thingsService) UpdateProfile(ctx context.Context, token string, profile Profile) error {
	ar := AuthorizeReq{
		Token:   token,
		Object:  profile.ID,
		Subject: ProfileSub,
		Action:  Editor,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return err
	}

	return ts.profiles.Update(ctx, profile)
}

func (ts *thingsService) ViewProfile(ctx context.Context, token, id string) (Profile, error) {
	profile, err := ts.profiles.RetrieveByID(ctx, id)
	if err != nil {
		return Profile{}, err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  profile.GroupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) ListProfiles(ctx context.Context, token string, pm PageMetadata) (ProfilesPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.profiles.RetrieveByAdmin(ctx, pm)
	}

	res, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ProfilesPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	grIDs, err := ts.groupCache.GroupIDsByMember(ctx, res.GetId())
	if err != nil {
		grIDs, err = ts.roles.RetrieveGroupIDsByMember(ctx, res.GetId())
		if err != nil {
			return ProfilesPage{}, err
		}
	}

	return ts.profiles.RetrieveByGroupIDs(ctx, grIDs, pm)
}

func (ts *thingsService) ViewProfileByThing(ctx context.Context, token, thID string) (Profile, error) {
	profile, err := ts.profiles.RetrieveByThing(ctx, thID)
	if err != nil {
		return Profile{}, err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  profile.GroupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := AuthorizeReq{
			Token:   token,
			Object:  id,
			Subject: ProfileSub,
			Action:  Editor,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return err
		}

		if things, err := ts.things.RetrieveByProfile(ctx, id, PageMetadata{}); err == nil {
			if things.PageMetadata.Total > 0 {
				return errors.ErrAuthorization
			}
		}

		if err := ts.profileCache.RemoveGroupID(ctx, id); err != nil {
			return err
		}
	}

	return ts.profiles.Remove(ctx, ids...)
}

func (ts *thingsService) GetPubConfByKey(ctx context.Context, thingKey string) (PubConfInfo, error) {
	thID, err := ts.thingCache.ID(ctx, thingKey)
	if err != nil {
		id, err := ts.things.RetrieveByKey(ctx, thingKey)
		if err != nil {
			return PubConfInfo{}, err
		}
		thID = id

		if err := ts.thingCache.Save(ctx, thingKey, thID); err != nil {
			return PubConfInfo{}, err
		}
	}

	profile, err := ts.profiles.RetrieveByThing(ctx, thID)
	if err != nil {
		return PubConfInfo{}, err
	}

	return PubConfInfo{PublisherID: thID, ProfileConfig: profile.Config}, nil
}

func (ts *thingsService) GetConfigByThingID(ctx context.Context, thingID string) (map[string]interface{}, error) {
	profile, err := ts.profiles.RetrieveByThing(ctx, thingID)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return profile.Config, nil
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
	case ProfileSub:
		profile, err := ts.profiles.RetrieveByID(ctx, ar.Object)
		if err != nil {
			return err
		}
		groupID = profile.GroupID
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

func (ts *thingsService) GetGroupIDByThingID(ctx context.Context, thingID string) (string, error) {
	thGrID, err := ts.thingCache.GroupID(ctx, thingID)
	if err != nil {
		th, err := ts.things.RetrieveByID(ctx, thingID)
		if err != nil {
			return "", err
		}
		thGrID = th.GroupID

		if err := ts.thingCache.SaveGroupID(ctx, th.ID, th.GroupID); err != nil {
			return "", err
		}
	}

	return thGrID, nil
}

func (ts *thingsService) Backup(ctx context.Context, token string) (Backup, error) {
	if err := ts.isAdmin(ctx, token); err != nil {
		return Backup{}, err
	}

	groups, err := ts.groups.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	groupsRoles, err := ts.roles.RetrieveAllRolesByGroup(ctx)
	if err != nil {
		return Backup{}, err
	}

	things, err := ts.things.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	profiles, err := ts.profiles.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		Things:     things,
		Profiles:   profiles,
		Groups:     groups,
		GroupRoles: groupsRoles,
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

	if _, err := ts.profiles.Save(ctx, backup.Profiles...); err != nil {
		return err
	}

	for _, g := range backup.GroupRoles {
		gm := GroupMember{
			MemberID: g.MemberID,
			GroupID:  g.GroupID,
			Role:     g.Role,
		}

		if err := ts.roles.SaveRolesByGroup(ctx, gm); err != nil {
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

func (ts *thingsService) ListProfilesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ProfilesPage, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return ProfilesPage{}, err
	}

	return ts.profiles.RetrieveByGroupIDs(ctx, []string{groupID}, pm)
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

func (ts *thingsService) canAccessOrg(ctx context.Context, token, orgID, subject, action string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Object:  orgID,
		Subject: subject,
		Action:  action,
	}

	if _, err := ts.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
