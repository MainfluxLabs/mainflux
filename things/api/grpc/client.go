// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ protomfx.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout        time.Duration
	getConnByKey   endpoint.Endpoint
	isChannelOwner endpoint.Endpoint
	canAccessGroup endpoint.Endpoint
	identify       endpoint.Endpoint
	getGroupsByIDs endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.ThingsServiceClient {
	svcName := "protomfx.ThingsService"

	return &grpcClient{
		timeout: timeout,
		getConnByKey: kitot.TraceClient(tracer, "get_conn_by_key")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetConnByKey",
			encodeGetConnByKeyRequest,
			decodeGetConnByKeyResponse,
			protomfx.ConnByKeyRes{},
		).Endpoint()),
		isChannelOwner: kitot.TraceClient(tracer, "is_channel_owner")(kitgrpc.NewClient(
			conn,
			svcName,
			"IsChannelOwner",
			encodeIsChannelOwner,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		canAccessGroup: kitot.TraceClient(tracer, "can_access_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanAccessGroup",
			encodeCanAccessGroup,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			protomfx.ThingID{},
		).Endpoint()),
		getGroupsByIDs: kitot.TraceClient(tracer, "get_groups_by_ids")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupsByIDs",
			encodeGetGroupsByIDsRequest,
			decodeGetGroupsByIDsResponse,
			protomfx.GroupsRes{},
		).Endpoint()),
	}
}

func (client grpcClient) GetConnByKey(ctx context.Context, req *protomfx.ConnByKeyReq, _ ...grpc.CallOption) (*protomfx.ConnByKeyRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	ar := connByKeyReq{
		key: req.GetKey(),
	}
	res, err := client.getConnByKey(ctx, ar)
	if err != nil {
		return nil, err
	}

	cr := res.(connByKeyRes)
	return &protomfx.ConnByKeyRes{ChannelID: cr.channelOD, ThingID: cr.thingID, Profile: cr.profile}, nil
}

func (client grpcClient) IsChannelOwner(ctx context.Context, req *protomfx.ChannelOwnerReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := channelOwnerReq{token: req.GetToken(), chanID: req.GetChanID()}
	res, err := client.isChannelOwner(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) CanAccessGroup(ctx context.Context, req *protomfx.AccessGroupReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := accessGroupReq{token: req.GetToken(), groupID: req.GetGroupID(), action: req.GetAction()}
	res, err := client.canAccessGroup(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) Identify(ctx context.Context, req *protomfx.Token, _ ...grpc.CallOption) (*protomfx.ThingID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identifyReq{key: req.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &protomfx.ThingID{Value: ir.id}, nil
}

func (client grpcClient) GetGroupsByIDs(ctx context.Context, req *protomfx.GroupsReq, _ ...grpc.CallOption) (*protomfx.GroupsRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupsByIDs(ctx, getGroupsByIDsReq{ids: req.GetIds()})
	if err != nil {
		return nil, err
	}

	gr := res.(getGroupsByIDsRes)
	return &protomfx.GroupsRes{Groups: gr.groups}, nil
}

func encodeGetConnByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(connByKeyReq)
	return &protomfx.ConnByKeyReq{Key: req.key}, nil
}

func encodeIsChannelOwner(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(channelOwnerReq)
	return &protomfx.ChannelOwnerReq{Token: req.token, ChanID: req.chanID}, nil
}

func encodeCanAccessGroup(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(accessGroupReq)
	return &protomfx.AccessGroupReq{Token: req.token, GroupID: req.groupID, Action: req.action}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &protomfx.Token{Value: req.key}, nil
}

func encodeGetGroupsByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(getGroupsByIDsReq)
	return &protomfx.GroupsReq{Ids: req.ids}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.ThingID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeGetConnByKeyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.ConnByKeyRes)
	return connByKeyRes{channelOD: res.ChannelID, thingID: res.ThingID, profile: res.Profile}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}

func decodeGetGroupsByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.GroupsRes)
	return getGroupsByIDsRes{groups: res.GetGroups()}, nil
}
