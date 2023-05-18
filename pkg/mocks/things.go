// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
	groups   map[string]things.Group
}

// NewThingsService returns mock implementation of things service
func NewThingsService(channels map[string]string, groups map[string]things.Group) mainflux.ThingsServiceClient {
	return &thingsServiceMock{channels, groups}
}

func (svc thingsServiceMock) CanAccessByKey(ctx context.Context, in *mainflux.AccessByKeyReq, opts ...grpc.CallOption) (*mainflux.ThingID, error) {
	token := in.GetToken()

	if token == "invalid" {
		return nil, errors.ErrAuthentication
	}

	if token == "" {
		return nil, errors.ErrAuthentication
	}

	if token == "token" {
		return nil, errors.ErrAuthorization
	}

	// Since there is no appropriate way to simulate internal server error,
	// we had to use this obscure approach. ErrorToken simulates gRPC
	// call which returns internal server error.
	if token == "unavailable" {
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &mainflux.ThingID{Value: token}, nil
}

func (svc thingsServiceMock) CanAccessByID(context.Context, *mainflux.AccessByIDReq, ...grpc.CallOption) (*empty.Empty, error) {
	panic("not implemented")
}

func (svc thingsServiceMock) IsChannelOwner(ctx context.Context, in *mainflux.ChannelOwnerReq, opts ...grpc.CallOption) (*empty.Empty, error) {
	if id, ok := svc.channels[in.GetOwner()]; ok {
		if id == in.ChanID {
			return nil, nil
		}
	}
	return nil, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(context.Context, *mainflux.Token, ...grpc.CallOption) (*mainflux.ThingID, error) {
	panic("not implemented")
}

func (svc thingsServiceMock) GetGroupsByIDs(ctx context.Context, req *mainflux.GroupsReq, opts ...grpc.CallOption) (*mainflux.GroupsRes, error) {
	var groups []*mainflux.Group
	for _, id := range req.Ids {
		if group, ok := svc.groups[id]; ok {
			groups = append(groups, &mainflux.Group{Id: group.ID, Name: group.Name})
		}
	}

	return &mainflux.GroupsRes{Groups: groups}, nil
}
