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
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	profiles map[string]things.Profile
	things   map[string]things.Thing
	groups   map[string]things.Group
}

// NewThingsServiceClient returns mock implementation of things service
func NewThingsServiceClient(profiles map[string]things.Profile, things map[string]things.Thing, groups map[string]things.Group) protomfx.ThingsServiceClient {
	return &thingsServiceMock{profiles, things, groups}
}

func (svc thingsServiceMock) GetPubConfByKey(_ context.Context, in *protomfx.PubConfByKeyReq, _ ...grpc.CallOption) (*protomfx.PubConfByKeyRes, error) {
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

	return &protomfx.PubConfByKeyRes{PublisherID: svc.things[key].ID}, nil
}

func (svc thingsServiceMock) GetConfigByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ConfigByThingIDRes, error) {
	panic("implement me")
}

func (svc thingsServiceMock) Authorize(_ context.Context, in *protomfx.AuthorizeReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	gr, ok := svc.groups[in.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	switch in.GetSubject() {
	case things.ThingSub:
		if th, ok := svc.things[in.GetToken()]; ok {
			if th.GroupID == gr.ID {
				return &empty.Empty{}, nil
			}
		}
	case things.ProfileSub:
		if pr, ok := svc.profiles[in.GetToken()]; ok {
			if pr.GroupID == gr.ID {
				return &empty.Empty{}, nil
			}
		}
	case things.GroupSub:
		if in.GetObject() == gr.ID {
			return &empty.Empty{}, nil
		}
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingAccessGroup(_ context.Context, req *protomfx.ThingAccessReq, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if th, ok := svc.things[req.GetKey()]; ok {
		if th.GroupID == req.GetId() {
			return &empty.Empty{}, nil
		}
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(_ context.Context, token *protomfx.Token, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	if th, ok := svc.things[token.GetValue()]; ok {
		return &protomfx.ThingID{Value: th.ID}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc thingsServiceMock) GetGroupsByIDs(_ context.Context, req *protomfx.GroupsReq, _ ...grpc.CallOption) (*protomfx.GroupsRes, error) {
	var groups []*protomfx.Group
	for _, id := range req.Ids {
		if group, ok := svc.groups[id]; ok {
			groups = append(groups, &protomfx.Group{Id: group.ID, Name: group.Name, Description: group.Description})
		}
	}

	return &protomfx.GroupsRes{Groups: groups}, nil
}

func (svc thingsServiceMock) GetGroupIDByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if th, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: th.GroupID}, nil
	}
	return nil, errors.ErrNotFound
}
