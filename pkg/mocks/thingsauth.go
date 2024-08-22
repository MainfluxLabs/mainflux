// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
	things   map[string]string
	groups   map[string]things.Group
}

// NewThingsServiceClient returns mock implementation of things service
func NewThingsServiceClient(channels map[string]string, things map[string]string, groups map[string]things.Group) protomfx.ThingsServiceClient {
	return &thingsServiceMock{channels, things, groups}
}

func (svc thingsServiceMock) GetConnByKey(ctx context.Context, in *protomfx.ConnByKeyReq, opts ...grpc.CallOption) (*protomfx.ConnByKeyRes, error) {
	key := in.GetKey()

	if key == "invalid" {
		return nil, errors.ErrAuthentication
	}

	if key == "" {
		return nil, errors.ErrAuthentication
	}

	if key == "token" {
		return nil, errors.ErrAuthorization
	}

	// Since there is no appropriate way to simulate internal server error,
	// we had to use this obscure approach. ErrorToken simulates gRPC
	// call which returns internal server error.
	if key == "unavailable" {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &protomfx.ConnByKeyRes{ChannelID: key, ThingID: svc.things[key]}, nil
}

func (svc thingsServiceMock) CanAccessGroup(ctx context.Context, in *protomfx.AccessGroupReq, opts ...grpc.CallOption) (*empty.Empty, error) {
	if id, ok := svc.things[in.GetToken()]; ok {
		if id == in.GroupID {
			return nil, nil
		}
	}
	return nil, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(ctx context.Context, token *protomfx.Token, opts ...grpc.CallOption) (*protomfx.ThingID, error) {
	if c, ok := svc.things[token.GetValue()]; ok {
		return &protomfx.ThingID{Value: c}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc thingsServiceMock) GetGroupsByIDs(ctx context.Context, req *protomfx.GroupsReq, opts ...grpc.CallOption) (*protomfx.GroupsRes, error) {
	var groups []*protomfx.Group
	for _, id := range req.Ids {
		if group, ok := svc.groups[id]; ok {
			groups = append(groups, &protomfx.Group{Id: group.ID, OwnerID: group.OwnerID, Name: group.Name, Description: group.Description})
		}
	}

	return &protomfx.GroupsRes{Groups: groups}, nil
}

func (svc thingsServiceMock) GetProfileByThing(ctx context.Context, in *protomfx.ThingID, opts ...grpc.CallOption) (*protomfx.ProfileByThingRes, error) {
	panic("implement me")
}

func (svc thingsServiceMock) GetGroupIDByThingID(ctx context.Context, in *protomfx.ThingID, opts ...grpc.CallOption) (*protomfx.GroupID, error) {
	if gr, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: gr}, nil
	}
	return nil, errors.ErrNotFound
}
