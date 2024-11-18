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
	profiles map[string]string
	things   map[string]string
	groups   map[string]things.Group
}

// NewThingsServiceClient returns mock implementation of things service
func NewThingsServiceClient(profiles map[string]string, things map[string]string, groups map[string]things.Group) protomfx.ThingsServiceClient {
	return &thingsServiceMock{profiles, things, groups}
}

func (svc thingsServiceMock) GetConnByKey(_ context.Context, in *protomfx.ConnByKeyReq, _ ...grpc.CallOption) (*protomfx.ConnByKeyRes, error) {
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

	return &protomfx.ConnByKeyRes{ProfileID: key, ThingID: svc.things[key]}, nil
}

func (svc thingsServiceMock) Authorize(_ context.Context, in *protomfx.AuthorizeReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	gr, ok := svc.groups[in.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	switch in.GetSubject() {
	case things.ThingSub:
		if id, ok := svc.things[in.GetToken()]; ok {
			if id == gr.ID {
				return &empty.Empty{}, nil
			}
		}
	case things.ProfileSub:
		if id, ok := svc.profiles[in.GetToken()]; ok {
			if id == gr.ID {
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

func (svc thingsServiceMock) Identify(_ context.Context, token *protomfx.Token, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	if c, ok := svc.things[token.GetValue()]; ok {
		return &protomfx.ThingID{Value: c}, nil
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

func (svc thingsServiceMock) GetConfigByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ConfigByThingIDRes, error) {
	panic("implement me")
}

func (svc thingsServiceMock) GetGroupIDByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if gr, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: gr}, nil
	}
	return nil, errors.ErrNotFound
}
