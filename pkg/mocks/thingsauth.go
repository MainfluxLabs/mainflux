// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
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

func (svc thingsServiceMock) GetPubConfByKey(_ context.Context, in *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.PubConfByKeyRes, error) {
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

	return &protomfx.PubConfByKeyRes{PublisherID: svc.things[key].ID}, nil
}

func (svc thingsServiceMock) GetConfigByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ConfigByThingIDRes, error) {
	panic("not implemented")
}

func (svc thingsServiceMock) CanUserAccessThing(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	th, ok := svc.things[req.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	if req.GetId() == th.ID {
		return &empty.Empty{}, nil
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessProfile(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	gr, ok := svc.groups[req.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	if pr, ok := svc.profiles[req.GetToken()]; ok {
		if pr.GroupID == gr.ID {
			return &empty.Empty{}, nil
		}
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessGroup(_ context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	gr, ok := svc.groups[req.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	if req.GetId() == gr.ID {
		return &empty.Empty{}, nil
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) CanUserAccessGroupThings(_ context.Context, req *protomfx.GroupThingsAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	gr, ok := svc.groups[req.GetToken()]
	if !ok {
		return &empty.Empty{}, errors.ErrAuthentication
	}

	for _, id := range req.ThingIds {
		if th, ok := svc.things[id]; ok {
			if th.GroupID != gr.ID {
				return &empty.Empty{}, errors.ErrAuthorization
			}
		}
	}

	return &empty.Empty{}, nil
}

func (svc thingsServiceMock) CanThingAccessGroup(_ context.Context, req *protomfx.ThingAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	if th, ok := svc.things[req.GetKey()]; ok {
		if th.GroupID == req.GetId() {
			return &empty.Empty{}, nil
		}
	}

	return &empty.Empty{}, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(_ context.Context, key *protomfx.ThingKey, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	if th, ok := svc.things[key.GetValue()]; ok {
		return &protomfx.ThingID{Value: th.ID}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc thingsServiceMock) GetGroupIDByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if th, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: th.GroupID}, nil
	}
	return nil, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetGroupIDByProfileID(_ context.Context, in *protomfx.ProfileID, _ ...grpc.CallOption) (*protomfx.GroupID, error) {
	if pr, ok := svc.profiles[in.GetValue()]; ok {
		return &protomfx.GroupID{Value: pr.GroupID}, nil
	}
	return nil, dbutil.ErrNotFound
}

func (svc thingsServiceMock) GetProfileIDByThingID(_ context.Context, in *protomfx.ThingID, _ ...grpc.CallOption) (*protomfx.ProfileID, error) {
	if th, ok := svc.things[in.GetValue()]; ok {
		return &protomfx.ProfileID{Value: th.ProfileID}, nil
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
