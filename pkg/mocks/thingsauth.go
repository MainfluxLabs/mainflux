// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	domain "github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ domain.ThingsClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	profiles map[string]domain.Profile
	things   map[string]domain.Thing
	groups   map[string]domain.Group
}

// NewThingsServiceClient returns mock implementation of things service
func NewThingsServiceClient(profiles map[string]domain.Profile, things map[string]domain.Thing, groups map[string]domain.Group) domain.ThingsClient {
	return &thingsServiceMock{profiles, things, groups}
}

func (svc thingsServiceMock) GetPubConfigByKey(_ context.Context, key domain.ThingKey) (domain.PubConfigInfo, error) {
	// Since there is no appropriate way to simulate internal server error,
	// we had to use this obscure approach. ErrorToken simulates gRPC
	// call which returns internal server error.
	if key.Value == "unavailable" {
		return domain.PubConfigInfo{}, status.Error(codes.Internal, "internal server error")
	}

	th, ok := svc.things[key.Value]
	if !ok {
		return domain.PubConfigInfo{}, errors.ErrAuthentication
	}

	return domain.PubConfigInfo{
		PublisherID: th.ID,
	}, nil
}

func (svc thingsServiceMock) GetConfigByThing(_ context.Context, _ string) (*domain.ProfileConfig, error) {
	return &domain.ProfileConfig{}, nil
}

func (svc thingsServiceMock) CanUserAccessThing(_ context.Context, req domain.UserAccessReq) error {
	th, ok := svc.things[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	if req.ID == th.ID {
		return nil
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessProfile(_ context.Context, req domain.UserAccessReq) error {
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

func (svc thingsServiceMock) CanUserAccessGroup(_ context.Context, req domain.UserAccessReq) error {
	gr, ok := svc.groups[req.Token]
	if !ok {
		return errors.ErrAuthentication
	}

	if req.ID == gr.ID {
		return nil
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingAccessGroup(_ context.Context, req domain.ThingAccessReq) error {
	if th, ok := svc.things[req.Value]; ok {
		if th.GroupID == req.ID {
			return nil
		}
	}

	return errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingCommand(_ context.Context, req domain.ThingCommandReq) error {
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

func (svc thingsServiceMock) CanThingGroupCommand(_ context.Context, req domain.ThingGroupCommandReq) error {
	publisher, ok := svc.things[req.PublisherID]
	if !ok {
		return errors.ErrAuthentication
	}

	if publisher.GroupID != req.GroupID {
		return errors.ErrAuthorization
	}

	return things.CanGroupCommand(publisher.Type)
}

func (svc thingsServiceMock) Identify(_ context.Context, key domain.ThingKey) (string, error) {
	if th, ok := svc.things[key.Value]; ok {
		return th.ID, nil
	}
	return "", errors.ErrAuthentication
}

func (svc thingsServiceMock) GetKeyByThingID(_ context.Context, thingID string) (domain.ThingKey, error) {
	if th, ok := svc.things[thingID]; ok {
		return domain.ThingKey{Value: th.Key}, nil
	}
	return domain.ThingKey{}, dbutil.ErrNotFound
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

func (svc thingsServiceMock) GetGroupIDsByOrg(_ context.Context, req domain.OrgAccessReq) ([]string, error) {
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

func (svc thingsServiceMock) CreateGroupMemberships(_ context.Context, _ ...domain.GroupMembership) error {
	return nil
}

func (svc thingsServiceMock) GetGroup(_ context.Context, groupID string) (domain.Group, error) {
	group, ok := svc.groups[groupID]
	if !ok {
		return domain.Group{}, dbutil.ErrNotFound
	}

	return domain.Group{
		ID:    group.ID,
		OrgID: group.OrgID,
		Name:  group.Name,
	}, nil
}
