// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
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

func (svc thingsServiceMock) GetPubConfigByKey(_ context.Context, in *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.PubConfigByKeyRes, error) {
	key := in.GetValue()

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

	return &protomfx.PubConfigByKeyRes{PublisherID: svc.things[key].ID}, nil
}

func (svc thingsServiceMock) GetConfigByThing(context.Context, *protomfx.ThingID, ...grpc.CallOption) (*protomfx.ConfigByThingRes, error) {
	panic("not implemented")
}

func (svc thingsServiceMock) CanUserAccessThing(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	th, ok := svc.things[req.GetToken()]
	if !ok {
		return &emptypb.Empty{}, errors.ErrAuthentication
	}

	if req.GetId() == th.ID {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessProfile(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	gr, ok := svc.groups[req.GetToken()]
	if !ok {
		return &emptypb.Empty{}, errors.ErrAuthentication
	}

	if pr, ok := svc.profiles[req.GetToken()]; ok {
		if pr.GroupID == gr.ID {
			return &emptypb.Empty{}, nil
		}
	}

	return &emptypb.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessGroup(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	gr, ok := svc.groups[req.GetToken()]
	if !ok {
		return &emptypb.Empty{}, errors.ErrAuthentication
	}

	if req.GetId() == gr.ID {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanThingAccessGroup(_ context.Context, req *protomfx.ThingAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	if th, ok := svc.things[req.GetKey()]; ok {
		if th.GroupID == req.GetId() {
			return &emptypb.Empty{}, nil
		}
	}

	return &emptypb.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(_ context.Context, key *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	if th, ok := svc.things[key.GetValue()]; ok {
		return &protomfx.ThingID{Value: th.ID}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc thingsServiceMock) GetKeyByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ThingKey, error) {
	if th, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.ThingKey{Value: th.Key}, nil
	}
	return nil, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDByThing(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if th, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: th.GroupID}, nil
	}
	return nil, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDByProfile(_ context.Context, in *protomfx.ProfileID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if pr, ok := svc.profiles[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: pr.GroupID}, nil
	}
	return nil, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDsByOrg(_ context.Context, in *protomfx.OrgAccessReq, _ ...grpc.CallOption) (*protomfx.GroupIDs, error) {
	var ids []string
	for _, g := range svc.groups {
		if g.OrgID == in.GetOrgId() {
			ids = append(ids, g.ID)
		}
	}
	return &protomfx.GroupIDs{Ids: ids}, nil
}

func (svc thingsServiceMock) GetThingIDsByProfile(_ context.Context, in *protomfx.ProfileID, _ ...grpc.CallOption) (*protomfx.ThingIDs, error) {
	var ids []string
	for _, t := range svc.things {
		if t.ProfileID == in.GetValue() {
			ids = append(ids, t.ID)
		}
	}
	return &protomfx.ThingIDs{Ids: ids}, nil
}

func (svc thingsServiceMock) CreateGroupMemberships(_ context.Context, in *protomfx.CreateGroupMembershipsReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (svc thingsServiceMock) GetGroup(_ context.Context, in *protomfx.GetGroupReq, _ ...grpc.CallOption) (*protomfx.Group, error) {
	group, ok := svc.groups[in.GetGroupID()]
	if !ok {
		return nil, dbutil.ErrNotFound
	}

	return &protomfx.Group{
		Id:    group.ID,
		OrgID: group.OrgID,
		Name:  group.Name,
	}, nil
}
