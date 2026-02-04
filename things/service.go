// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	Viewer = "viewer"
	Editor = "editor"
	Admin  = "admin"
	Owner  = "owner"
)

var (
	ErrProfileAssigned = errors.New("profile currently assigned to thing(s)")
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings adds things to the user identified by the token.
	// The group ID for each thing is assigned based on the provided profile ID.
	CreateThings(ctx context.Context, token, profileID string, things ...Thing) ([]Thing, error)

	// UpdateThing updates the Thing identified by the provided ID, as the user authenticated by 'token',
	// who must possess required permissions in the Thing's belonging Group.
	UpdateThing(ctx context.Context, token string, thing Thing) error

	// UpdateThingGroupAndProfile updates the Thing's belonging Profile or Group.
	UpdateThingGroupAndProfile(ctx context.Context, token string, thing Thing) error

	// UpdateThingsMetadata updates the things metadata identified by the provided IDs, that
	// belongs to the user identified by the provided token.
	UpdateThingsMetadata(ctx context.Context, token string, things ...Thing) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(ctx context.Context, token, id string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(ctx context.Context, token string, pm apiutil.PageMetadata) (ThingsPage, error)

	// ListThingsByOrg retrieves page of things that belong to an org identified by ID.
	ListThingsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (ThingsPage, error)

	// ListThingsByProfile retrieves data about subset of things that are
	// connected or not connected to specified profile and belong to the user identified by
	// the provided key.
	ListThingsByProfile(ctx context.Context, token, prID string, pm apiutil.PageMetadata) (ThingsPage, error)

	// RemoveThings removes the things identified with the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveThings(ctx context.Context, token string, id ...string) error

	// CreateProfiles adds profiles to the user identified by the token.
	// The group ID is assigned to each profile.
	CreateProfiles(ctx context.Context, token, groupID string, profiles ...Profile) ([]Profile, error)

	// UpdateProfile updates the profile identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateProfile(ctx context.Context, token string, profile Profile) error

	// ViewProfile retrieves data about the profile identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewProfile(ctx context.Context, token, id string) (Profile, error)

	// ListProfiles retrieves data about subset of profiles that belongs to the
	// user identified by the provided key.
	ListProfiles(ctx context.Context, token string, pm apiutil.PageMetadata) (ProfilesPage, error)

	// ListProfilesByOrg retrieves page of profiles that belong to an org identified by ID.
	ListProfilesByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (ProfilesPage, error)

	// ViewProfileByThing retrieves data about profile that have
	// specified thing connected or not connected to it and belong to the user identified by
	// the provided key.
	ViewProfileByThing(ctx context.Context, token, thID string) (Profile, error)

	// ViewMetadataByKey retrieves metadata about the thing identified by the given key.
	ViewMetadataByKey(ctx context.Context, key ThingKey) (Metadata, error)

	// RemoveProfiles removes the things identified by the provided IDs, that
	// belongs to the user identified by the provided key.
	RemoveProfiles(ctx context.Context, token string, ids ...string) error

	// GetPubConfigByKey retrieves the thing ID and its corresponding profile config
	// associated with the provided key.
	GetPubConfigByKey(ctx context.Context, key ThingKey) (PubConfigInfo, error)

	// GetConfigByThing returns profile config for given thing ID.
	GetConfigByThing(ctx context.Context, thingID string) (map[string]any, error)

	// CanUserAccessThing determines whether a user has access to a thing.
	CanUserAccessThing(ctx context.Context, req UserAccessReq) error

	// CanUserAccessProfile determines whether a user has access to a profile.
	CanUserAccessProfile(ctx context.Context, req UserAccessReq) error

	// CanUserAccessGroup determines whether a user has access to a group.
	CanUserAccessGroup(ctx context.Context, req UserAccessReq) error

	// CanThingAccessGroup determines whether a given thing has access to a group with a key.
	CanThingAccessGroup(ctx context.Context, req ThingAccessReq) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key ThingKey) (string, error)

	// GetKeyByThingID returns a thing's key for given thing ID.
	GetKeyByThingID(ctx context.Context, thingID string) (ThingKey, error)

	// GetGroupIDByThing returns a thing's group ID for given thing ID.
	GetGroupIDByThing(ctx context.Context, thingID string) (string, error)

	// GetGroupIDByProfile returns a profile's group ID for given profile ID.
	GetGroupIDByProfile(ctx context.Context, profileID string) (string, error)

	// GetGroupIDsByOrg returns all group IDs belonging to an org.
	GetGroupIDsByOrg(ctx context.Context, orgID string, token string) ([]string, error)

	// UpdateExternalKey updates the external key of the Thing identified by `thingID`. The authenticated user must have Editor rights within the Thing's belonging Group.
	UpdateExternalKey(ctx context.Context, token, key, thingID string) error

	// RemoveExternalKey removes the external thing key of the Thing identified by `thingID`.
	// The authenticated user must have Editor rights within the Thing's belonging Group.
	RemoveExternalKey(ctx context.Context, token, thingID string) error

	// Backup retrieves all things, profiles, groups, and groups memberships for all users. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds things, profiles, groups, and groups memberships from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error

	// GetThingIDsByProfile returns the IDs of all things associated with the given profile ID.
	GetThingIDsByProfile(ctx context.Context, profileID string) ([]string, error)

	Groups

	GroupMemberships
}

type Backup struct {
	Things           []Thing
	Profiles         []Profile
	Groups           []Group
	GroupMemberships []GroupMembership
}

type UserAccessReq struct {
	Token  string
	ID     string
	Action string
}

type ThingAccessReq struct {
	ThingKey
	ID string
}

type PubConfigInfo struct {
	PublisherID   string
	ProfileConfig map[string]any
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth             protomfx.AuthServiceClient
	users            protomfx.UsersServiceClient
	things           ThingRepository
	profiles         ProfileRepository
	groups           GroupRepository
	groupMemberships GroupMembershipsRepository
	profileCache     ProfileCache
	thingCache       ThingCache
	groupCache       GroupCache
	idProvider       uuid.IDProvider
	email            Emailer
}

// New instantiates the things service implementation.
func New(auth protomfx.AuthServiceClient, users protomfx.UsersServiceClient, things ThingRepository, profiles ProfileRepository,
	groups GroupRepository, groupMemberships GroupMembershipsRepository,
	pcache ProfileCache, tcache ThingCache, gcache GroupCache, idp uuid.IDProvider,
	emailer Emailer) Service {
	return &thingsService{
		auth:             auth,
		users:            users,
		things:           things,
		profiles:         profiles,
		groups:           groups,
		groupMemberships: groupMemberships,
		profileCache:     pcache,
		thingCache:       tcache,
		groupCache:       gcache,
		idProvider:       idp,
		email:            emailer,
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token, profileID string, things ...Thing) ([]Thing, error) {
	groupID, err := ts.getGroupIDByProfile(ctx, profileID)
	if err != nil {
		return []Thing{}, err
	}

	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Editor,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return nil, err
	}

	ths := []Thing{}
	for _, thing := range things {
		thing.ProfileID = profileID
		thing.GroupID = groupID
		if thing.ID == "" {
			id, err := ts.idProvider.ID()
			if err != nil {
				return []Thing{}, err
			}
			thing.ID = id
		}

		if thing.Key == "" {
			key, err := ts.idProvider.ID()

			if err != nil {
				return []Thing{}, err
			}
			thing.Key = key
		}

		ths = append(ths, thing)
	}

	return ts.things.Save(ctx, ths...)
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     thing.ID,
		Action: Editor,
	}

	if err := ts.CanUserAccessThing(ctx, ar); err != nil {
		return err
	}

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) UpdateThingGroupAndProfile(ctx context.Context, token string, thing Thing) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     thing.ID,
		Action: Editor,
	}

	if err := ts.CanUserAccessThing(ctx, ar); err != nil {
		return err
	}

	if err := ts.canAccessGroup(ctx, token, thing.GroupID, Editor); err != nil {
		return err
	}

	prGrID, err := ts.getGroupIDByProfile(ctx, thing.ProfileID)
	if err != nil {
		return err
	}

	if prGrID != thing.GroupID {
		return errors.ErrAuthorization
	}

	return ts.things.UpdateGroupAndProfile(ctx, thing)
}

func (ts *thingsService) UpdateThingsMetadata(ctx context.Context, token string, things ...Thing) error {
	for _, thing := range things {
		ar := UserAccessReq{
			Token:  token,
			ID:     thing.ID,
			Action: Editor,
		}

		if err := ts.CanUserAccessThing(ctx, ar); err != nil {
			return err
		}

		th, err := ts.things.RetrieveByID(ctx, thing.ID)
		if err != nil {
			return err
		}

		for k, v := range thing.Metadata {
			th.Metadata[k] = v
		}

		if err := ts.things.Update(ctx, th); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     id,
		Action: Viewer,
	}
	if err := ts.CanUserAccessThing(ctx, ar); err != nil {
		return Thing{}, err
	}

	thing, err := ts.things.RetrieveByID(ctx, id)
	if err != nil {
		return Thing{}, err
	}

	return thing, nil
}

func (ts *thingsService) ViewMetadataByKey(ctx context.Context, key ThingKey) (Metadata, error) {
	thingID, err := ts.Identify(ctx, key)
	if err != nil {
		return Metadata{}, err
	}

	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return Metadata{}, err
	}

	return thing.Metadata, nil
}

func (ts *thingsService) ListThings(ctx context.Context, token string, pm apiutil.PageMetadata) (ThingsPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.things.RetrieveAll(ctx, pm)
	}

	res, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ThingsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	grIDs, err := ts.getGroupIDsByMember(ctx, res.GetId())
	if err != nil {
		return ThingsPage{}, err
	}

	return ts.things.RetrieveByGroups(ctx, grIDs, pm)
}

func (ts *thingsService) ListThingsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (ThingsPage, error) {
	grIDs, err := ts.GetGroupIDsByOrg(ctx, orgID, token)
	if err != nil {
		return ThingsPage{}, err
	}

	return ts.things.RetrieveByGroups(ctx, grIDs, pm)
}

func (ts *thingsService) ListThingsByProfile(ctx context.Context, token, prID string, pm apiutil.PageMetadata) (ThingsPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     prID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessProfile(ctx, ar); err != nil {
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
		ar := UserAccessReq{
			Token:  token,
			ID:     id,
			Action: Editor,
		}
		if err := ts.CanUserAccessThing(ctx, ar); err != nil {
			return err
		}

		if err := ts.thingCache.RemoveThing(ctx, id); err != nil {
			return err
		}

		if err := ts.thingCache.RemoveGroup(ctx, id); err != nil {
			return err
		}
	}

	if err := ts.things.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) CreateProfiles(ctx context.Context, token, groupID string, profiles ...Profile) ([]Profile, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Editor,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return nil, err
	}

	for i := range profiles {
		profiles[i].GroupID = groupID
		if profiles[i].ID == "" {
			prID, err := ts.idProvider.ID()
			if err != nil {
				return []Profile{}, err
			}
			profiles[i].ID = prID
		}
	}

	prs, err := ts.profiles.Save(ctx, profiles...)
	if err != nil {
		return []Profile{}, err
	}

	return prs, nil
}

func (ts *thingsService) UpdateProfile(ctx context.Context, token string, profile Profile) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     profile.ID,
		Action: Editor,
	}
	if err := ts.CanUserAccessProfile(ctx, ar); err != nil {
		return err
	}

	return ts.profiles.Update(ctx, profile)
}

func (ts *thingsService) ViewProfile(ctx context.Context, token, id string) (Profile, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     id,
		Action: Viewer,
	}
	if err := ts.CanUserAccessProfile(ctx, ar); err != nil {
		return Profile{}, err
	}

	profile, err := ts.profiles.RetrieveByID(ctx, id)
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) ListProfiles(ctx context.Context, token string, pm apiutil.PageMetadata) (ProfilesPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.profiles.RetrieveAll(ctx, pm)
	}

	res, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ProfilesPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	grIDs, err := ts.getGroupIDsByMember(ctx, res.GetId())
	if err != nil {
		return ProfilesPage{}, err
	}

	return ts.profiles.RetrieveByGroups(ctx, grIDs, pm)
}

func (ts *thingsService) ListProfilesByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (ProfilesPage, error) {
	grIDs, err := ts.GetGroupIDsByOrg(ctx, orgID, token)
	if err != nil {
		return ProfilesPage{}, err
	}

	return ts.profiles.RetrieveByGroups(ctx, grIDs, pm)
}

func (ts *thingsService) ViewProfileByThing(ctx context.Context, token, thID string) (Profile, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     thID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessThing(ctx, ar); err != nil {
		return Profile{}, err
	}

	profile, err := ts.profiles.RetrieveByThing(ctx, thID)
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (ts *thingsService) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := UserAccessReq{
			Token:  token,
			ID:     id,
			Action: Editor,
		}

		if err := ts.CanUserAccessProfile(ctx, ar); err != nil {
			return err
		}

		if err := ts.profileCache.RemoveGroup(ctx, id); err != nil {
			return err
		}
	}

	return ts.profiles.Remove(ctx, ids...)
}

func (ts *thingsService) GetPubConfigByKey(ctx context.Context, key ThingKey) (PubConfigInfo, error) {
	thID, err := ts.thingCache.ID(ctx, key)
	if err != nil {
		id, err := ts.things.RetrieveByKey(ctx, key)
		if err != nil {
			return PubConfigInfo{}, err
		}
		thID = id

		if err := ts.thingCache.Save(ctx, key, thID); err != nil {
			return PubConfigInfo{}, err
		}
	}

	profile, err := ts.profiles.RetrieveByThing(ctx, thID)
	if err != nil {
		return PubConfigInfo{}, err
	}

	res := PubConfigInfo{
		PublisherID:   thID,
		ProfileConfig: profile.Config,
	}

	return res, nil
}

func (ts *thingsService) GetConfigByThing(ctx context.Context, thingID string) (map[string]any, error) {
	profile, err := ts.profiles.RetrieveByThing(ctx, thingID)
	if err != nil {
		return map[string]any{}, err
	}

	return profile.Config, nil
}

func (ts *thingsService) CanUserAccessThing(ctx context.Context, req UserAccessReq) error {
	grID, err := ts.getGroupIDByThing(ctx, req.ID)
	if err != nil {
		return err
	}

	return ts.canAccessGroup(ctx, req.Token, grID, req.Action)
}

func (ts *thingsService) CanUserAccessProfile(ctx context.Context, req UserAccessReq) error {
	grID, err := ts.getGroupIDByProfile(ctx, req.ID)
	if err != nil {
		return err
	}

	return ts.canAccessGroup(ctx, req.Token, grID, req.Action)
}

func (ts *thingsService) CanUserAccessGroup(ctx context.Context, req UserAccessReq) error {
	if _, err := ts.groups.RetrieveByID(ctx, req.ID); err != nil {
		return err
	}
	return ts.canAccessGroup(ctx, req.Token, req.ID, req.Action)
}

func (ts *thingsService) CanThingAccessGroup(ctx context.Context, req ThingAccessReq) error {
	thID, err := ts.Identify(ctx, req.ThingKey)
	if err != nil {
		return err
	}

	grID, err := ts.getGroupIDByThing(ctx, thID)
	if err != nil {
		return err
	}

	if grID != req.ID {
		return errors.ErrAuthorization
	}

	return nil
}

func (ts *thingsService) Identify(ctx context.Context, key ThingKey) (string, error) {
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

func (ts *thingsService) UpdateExternalKey(ctx context.Context, token, key, thingID string) error {
	accessReq := UserAccessReq{
		Token:  token,
		ID:     thingID,
		Action: Editor,
	}

	if err := ts.CanUserAccessThing(ctx, accessReq); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return err
	}

	if err := ts.things.UpdateExternalKey(ctx, key, thingID); err != nil {
		return err
	}

	if err := ts.thingCache.RemoveKey(ctx, ThingKey{Type: KeyTypeExternal, Value: thing.ExternalKey}); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) RemoveExternalKey(ctx context.Context, token, thingID string) error {
	accessReq := UserAccessReq{
		Token:  token,
		ID:     thingID,
		Action: Editor,
	}

	if err := ts.CanUserAccessThing(ctx, accessReq); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return err
	}

	if err := ts.things.RemoveExternalKey(ctx, thingID); err != nil {
		return err
	}

	if err := ts.thingCache.RemoveKey(ctx, ThingKey{Type: KeyTypeExternal, Value: thing.ExternalKey}); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) GetKeyByThingID(ctx context.Context, thingID string) (ThingKey, error) {
	thing, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return ThingKey{}, err
	}

	return ThingKey{
		Value: thing.Key,
		Type:  KeyTypeInternal,
	}, nil
}

func (ts *thingsService) GetGroupIDByThing(ctx context.Context, thingID string) (string, error) {
	return ts.getGroupIDByThing(ctx, thingID)
}

func (ts *thingsService) GetGroupIDByProfile(ctx context.Context, profileID string) (string, error) {
	return ts.getGroupIDByProfile(ctx, profileID)
}

func (ts *thingsService) Backup(ctx context.Context, token string) (Backup, error) {
	if err := ts.isAdmin(ctx, token); err != nil {
		return Backup{}, err
	}

	groups, err := ts.groups.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	groupMemberships, err := ts.groupMemberships.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	things, err := ts.things.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	profiles, err := ts.profiles.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		Things:           things,
		Profiles:         profiles,
		Groups:           groups,
		GroupMemberships: groupMemberships,
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

	if _, err := ts.profiles.Save(ctx, backup.Profiles...); err != nil {
		return err
	}

	if _, err := ts.things.Save(ctx, backup.Things...); err != nil {
		return err
	}

	for _, g := range backup.GroupMemberships {
		gm := GroupMembership{
			MemberID: g.MemberID,
			GroupID:  g.GroupID,
			Role:     g.Role,
		}

		if err := ts.groupMemberships.Save(ctx, gm); err != nil {
			return err
		}
	}

	return nil
}

func getTimestamp() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (ts *thingsService) ListThingsByGroup(ctx context.Context, token string, groupID string, pm apiutil.PageMetadata) (ThingsPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return ThingsPage{}, err
	}

	return ts.things.RetrieveByGroups(ctx, []string{groupID}, pm)
}

func (ts *thingsService) ListProfilesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (ProfilesPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return ProfilesPage{}, err
	}

	return ts.profiles.RetrieveByGroups(ctx, []string{groupID}, pm)
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

func (ts *thingsService) getGroupIDByThing(ctx context.Context, thID string) (string, error) {
	grID, err := ts.thingCache.ViewGroup(ctx, thID)
	if err != nil {
		th, err := ts.things.RetrieveByID(ctx, thID)
		if err != nil {
			return "", err
		}
		grID = th.GroupID

		if err := ts.thingCache.SaveGroup(ctx, th.ID, th.GroupID); err != nil {
			return "", err
		}
	}

	return grID, nil
}

func (ts *thingsService) getGroupIDByProfile(ctx context.Context, prID string) (string, error) {
	grID, err := ts.profileCache.ViewGroup(ctx, prID)
	if err != nil {
		pr, err := ts.profiles.RetrieveByID(ctx, prID)
		if err != nil {
			return "", err
		}
		grID = pr.GroupID

		if err := ts.profileCache.SaveGroup(ctx, pr.ID, pr.GroupID); err != nil {
			return "", err
		}
	}

	return grID, nil
}

func (ts *thingsService) getGroupIDsByMember(ctx context.Context, memberID string) ([]string, error) {
	grIDs, err := ts.groupCache.RetrieveGroupIDsByMember(ctx, memberID)
	if err != nil {
		grIDs, err = ts.groupMemberships.RetrieveGroupIDsByMember(ctx, memberID)
		if err != nil {
			return []string{}, err
		}
	}
	return grIDs, nil
}

func (ts *thingsService) GetGroupIDsByOrg(ctx context.Context, orgID string, token string) ([]string, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.groups.RetrieveIDsByOrg(ctx, orgID)
	}

	if err := ts.canAccessOrg(ctx, token, orgID, auth.OrgSub, Viewer); err != nil {
		return []string{}, err
	}

	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return nil, err
	}

	return ts.groups.RetrieveIDsByOrgMembership(ctx, orgID, user.GetId())
}

func (ts *thingsService) GetThingIDsByProfile(ctx context.Context, profileID string) ([]string, error) {
	page, err := ts.things.RetrieveByProfile(ctx, profileID, apiutil.PageMetadata{})
	if err != nil {
		return []string{}, err
	}

	var thingIDs []string
	for _, t := range page.Things {
		thingIDs = append(thingIDs, t.ID)
	}
	return thingIDs, nil
}
