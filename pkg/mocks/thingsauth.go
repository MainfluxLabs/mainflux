// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ domainthings.Client = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	profiles map[string]domainthings.Profile
	things   map[string]domainthings.Thing
	groups   map[string]domainthings.Group
}

// NewThingsServiceClient returns mock implementation of things service
func NewThingsServiceClient(profiles map[string]domainthings.Profile, things map[string]domainthings.Thing, groups map[string]domainthings.Group) domainthings.Client {
	return &thingsServiceMock{profiles, things, groups}
}

func (svc thingsServiceMock) GetPubConfigByKey(_ context.Context, key domainthings.ThingKey) (domainthings.PubConfigInfo, error) {
	if key.Value == "invalid" {
		return domainthings.PubConfigInfo{}, errors.ErrAuthentication
	}

	if key.Value == "" {
		return domainthings.PubConfigInfo{}, errors.ErrAuthentication
	}

	if key.Value == "token" {
		return domainthings.PubConfigInfo{}, errors.ErrAuthorization
	}

	if key.Value == "unavailable" {
		return domainthings.PubConfigInfo{}, status.Error(codes.Internal, "internal server error")
	}

	if th, ok := svc.things[key.Value]; ok {
		return domainthings.PubConfigInfo{PublisherID: th.ID}, nil
	}
	// When things map is nil/empty, use key as PublisherID (old mock behavior for tests)
	if svc.things == nil || len(svc.things) == 0 {
		return domainthings.PubConfigInfo{PublisherID: key.Value}, nil
	}
	return domainthings.PubConfigInfo{}, errors.ErrAuthentication
}

func (svc thingsServiceMock) GetConfigByThing(_ context.Context, _ string) (domainthings.Config, error) {
	return domainthings.Config{}, nil
}

func (svc thingsServiceMock) CanUserAccessThing(_ context.Context, req domainthings.UserAccessReq) error {
	th, ok := svc.things[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	if req.ID == th.ID {
		return nil
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessProfile(_ context.Context, req domainthings.UserAccessReq) error {
	gr, ok := svc.groups[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	if pr, ok := svc.profiles[req.Token]; ok {
		if pr.GroupID == gr.ID {
			return nil
		}
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessGroup(_ context.Context, req domainthings.UserAccessReq) error {
	gr, ok := svc.groups[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	if req.ID == gr.ID {
		return nil
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingAccessGroup(_ context.Context, req domainthings.ThingAccessReq) error {
	if th, ok := svc.things[req.Value]; ok {
		if th.GroupID == req.ID {
			return nil
		}
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingCommand(_ context.Context, req domainthings.ThingCommandReq) error {
	publisher, ok := svc.things[req.PublisherID]
	if !ok {
		return errors.ErrAuthentication
	}

	recipient, ok := svc.things[req.RecipientID]
	if !ok {
		return errors.ErrAuthentication
	}

	if publisher.GroupID != recipient.GroupID {
		return errors.ErrAuthorization
	}

	return things.CanCommand(publisher.Type, recipient.Type)
}

func (svc thingsServiceMock) CanThingGroupCommand(_ context.Context, req domainthings.ThingGroupCommandReq) error {
	publisher, ok := svc.things[req.PublisherID]
	if !ok {
		return errors.ErrAuthentication
	}

	if publisher.GroupID != req.GroupID {
		return errors.ErrAuthorization
	}

	return things.CanGroupCommand(publisher.Type)
}

func (svc thingsServiceMock) Identify(_ context.Context, key domainthings.ThingKey) (string, error) {
	if th, ok := svc.things[key.Value]; ok {
		return th.ID, nil
	}
	return "", errors.ErrAuthentication
}

func (svc thingsServiceMock) GetKeyByThingID(_ context.Context, thingID string) (domainthings.ThingKey, error) {
	if th, ok := svc.things[thingID]; ok {
		return domainthings.ThingKey{Value: th.Key}, nil
	}
	return domainthings.ThingKey{}, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDByThing(_ context.Context, thingID string) (string, error) {
	if th, ok := svc.things[thingID]; ok {
		return th.GroupID, nil
	}
	return "", dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDByProfile(_ context.Context, profileID string) (string, error) {
	if pr, ok := svc.profiles[profileID]; ok {
		return pr.GroupID, nil
	}
	return "", dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDsByOrg(_ context.Context, req domainthings.OrgAccessReq) ([]string, error) {
	var ids []string
	for _, g := range svc.groups {
		if g.OrgID == req.OrgID {
			ids = append(ids, g.ID)
		}
	}
	return ids, nil
}

func (svc thingsServiceMock) GetThingIDsByProfile(_ context.Context, profileID string) ([]string, error) {
	var ids []string
	for _, t := range svc.things {
		if t.ProfileID == profileID {
			ids = append(ids, t.ID)
		}
	}
	return ids, nil
}

func (svc thingsServiceMock) CreateGroupMemberships(_ context.Context, _ ...domainthings.GroupMembership) error {
	return nil
}

func (svc thingsServiceMock) GetGroup(_ context.Context, groupID string) (domainthings.Group, error) {
	group, ok := svc.groups[groupID]
	if !ok {
		return domainthings.Group{}, dbutil.ErrNotFound
	}

	return domainthings.Group{
		ID:    group.ID,
		OrgID: group.OrgID,
		Name:  group.Name,
	}, nil
}
