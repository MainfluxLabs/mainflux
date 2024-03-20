// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ mainflux.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout        time.Duration
	getConnByKey   endpoint.Endpoint
	isChannelOwner endpoint.Endpoint
	isThingOwner   endpoint.Endpoint
	identify       endpoint.Endpoint
	getGroupsByIDs endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) mainflux.ThingsServiceClient {
	svcName := "mainflux.ThingsService"

	return &grpcClient{
		timeout: timeout,
		getConnByKey: kitot.TraceClient(tracer, "get_conn_by_key")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetConnByKey",
			encodeGetConnByKeyRequest,
			decodeGetConnByKeyResponse,
			mainflux.ConnByKeyRes{},
		).Endpoint()),
		isChannelOwner: kitot.TraceClient(tracer, "is_channel_owner")(kitgrpc.NewClient(
			conn,
			svcName,
			"IsChannelOwner",
			encodeIsChannelOwner,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		isThingOwner: kitot.TraceClient(tracer, "is_thing_owner")(kitgrpc.NewClient(
			conn,
			svcName,
			"IsThingOwner",
			encodeIsThingOwner,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			mainflux.ThingID{},
		).Endpoint()),
		getGroupsByIDs: kitot.TraceClient(tracer, "get_groups_by_ids")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupsByIDs",
			encodeGetGroupsByIDsRequest,
			decodeGetGroupsByIDsResponse,
			mainflux.GroupsRes{},
		).Endpoint()),
	}
}

func (client grpcClient) GetConnByKey(ctx context.Context, req *mainflux.ConnByKeyReq, _ ...grpc.CallOption) (*mainflux.ConnByKeyRes, error) {
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
	return &mainflux.ConnByKeyRes{ChannelID: cr.channelOD, ThingID: cr.thingID, Profile: cr.profile}, nil
}

func (client grpcClient) IsChannelOwner(ctx context.Context, req *mainflux.ChannelOwnerReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := channelOwnerReq{token: req.GetToken(), chanID: req.GetChanID()}
	res, err := client.isChannelOwner(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) IsThingOwner(ctx context.Context, req *mainflux.ThingOwnerReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	ar := thingOwnerReq{token: req.GetToken(), thingID: req.GetThingID()}
	res, err := client.isThingOwner(ctx, ar)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) Identify(ctx context.Context, req *mainflux.Token, _ ...grpc.CallOption) (*mainflux.ThingID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identifyReq{key: req.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.ThingID{Value: ir.id}, nil
}

func (client grpcClient) GetGroupsByIDs(ctx context.Context, req *mainflux.GroupsReq, _ ...grpc.CallOption) (*mainflux.GroupsRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupsByIDs(ctx, getGroupsByIDsReq{ids: req.GetIds()})
	if err != nil {
		return nil, err
	}

	gr := res.(getGroupsByIDsRes)
	return &mainflux.GroupsRes{Groups: gr.groups}, nil
}

func encodeGetConnByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(connByKeyReq)
	return &mainflux.ConnByKeyReq{Key: req.key}, nil
}

func encodeIsChannelOwner(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(channelOwnerReq)
	return &mainflux.ChannelOwnerReq{Token: req.token, ChanID: req.chanID}, nil
}

func encodeIsThingOwner(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(thingOwnerReq)
	return &mainflux.ThingOwnerReq{Token: req.token, ThingID: req.thingID}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &mainflux.Token{Value: req.key}, nil
}

func encodeGetGroupsByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(getGroupsByIDsReq)
	return &mainflux.GroupsReq{Ids: req.ids}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.ThingID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeGetConnByKeyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.ConnByKeyRes)
	return connByKeyRes{channelOD: res.ChannelID, thingID: res.ThingID, profile: res.Profile}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}

func decodeGetGroupsByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.GroupsRes)
	return getGroupsByIDsRes{groups: res.GetGroups()}, nil
}
