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
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout              time.Duration
	getPubConfByKey      endpoint.Endpoint
	getConfigByThingID   endpoint.Endpoint
	canUserAccessThing   endpoint.Endpoint
	canUserAccessProfile endpoint.Endpoint
	canUserAccessGroup   endpoint.Endpoint
	canThingAccessGroup  endpoint.Endpoint
	identify             endpoint.Endpoint
	getGroupIDByThingID  endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.ThingsServiceClient {
	svcName := "protomfx.ThingsService"

	return &grpcClient{
		timeout: timeout,
		getPubConfByKey: kitot.TraceClient(tracer, "get_pub_conf_by_key")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetPubConfByKey",
			encodeGetPubConfByKeyRequest,
			decodeGetPubConfByKeyResponse,
			protomfx.PubConfByKeyRes{},
		).Endpoint()),
		getConfigByThingID: kitot.TraceClient(tracer, "get_config_by_thing_id")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetConfigByThingID",
			encodeGetConfigByThingIDRequest,
			decodeGetConfigByThingIDResponse,
			protomfx.ConfigByThingIDRes{},
		).Endpoint()),
		canUserAccessThing: kitot.TraceClient(tracer, "can_user_access_thing")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessThing",
			encodeUserAccessThingRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		canUserAccessProfile: kitot.TraceClient(tracer, "can_user_access_profile")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessProfile",
			encodeUserAccessProfileRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		canUserAccessGroup: kitot.TraceClient(tracer, "can_user_access_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanUserAccessGroup",
			encodeUserAccessGroupRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		canThingAccessGroup: kitot.TraceClient(tracer, "can_thing_access_group")(kitgrpc.NewClient(
			conn,
			svcName,
			"CanThingAccessGroup",
			encodeThingAccessGroupRequest,
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
		getGroupIDByThingID: kitot.TraceClient(tracer, "get_group_id_by_thing_id")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetGroupIDByThingID",
			encodeGetGroupIDByThingIDRequest,
			decodeGetGroupIDByThingIDResponse,
			protomfx.GroupID{},
		).Endpoint()),
	}
}

func (client grpcClient) GetPubConfByKey(ctx context.Context, req *protomfx.PubConfByKeyReq, _ ...grpc.CallOption) (*protomfx.PubConfByKeyRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	ar := pubConfByKeyReq{
		key: req.GetKey(),
	}
	res, err := client.getPubConfByKey(ctx, ar)
	if err != nil {
		return nil, err
	}

	pc := res.(pubConfByKeyRes)
	return &protomfx.PubConfByKeyRes{PublisherID: pc.publisherID, ProfileConfig: pc.profileConfig}, nil
}

func (client grpcClient) GetConfigByThingID(ctx context.Context, req *protomfx.ThingID, opts ...grpc.CallOption) (*protomfx.ConfigByThingIDRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	res, err := client.getConfigByThingID(ctx, configByThingIDReq{thingID: req.GetValue()})
	if err != nil {
		return nil, err
	}
	c := res.(configByThingIDRes)
	return &protomfx.ConfigByThingIDRes{Config: c.config}, nil
}

func (client grpcClient) CanUserAccessThing(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessThingReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessThing(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) CanUserAccessProfile(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessProfileReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessProfile(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) CanUserAccessGroup(ctx context.Context, req *protomfx.UserAccessReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	r := userAccessGroupReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}
	res, err := client.canUserAccessGroup(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func (client grpcClient) CanThingAccessGroup(ctx context.Context, req *protomfx.ThingAccessReq, _ ...grpc.CallOption) (*empty.Empty, error) {
	r := thingAccessGroupReq{key: req.GetKey(), id: req.GetId()}
	res, err := client.canThingAccessGroup(ctx, r)
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

func (client grpcClient) GetGroupIDByThingID(ctx context.Context, req *protomfx.ThingID, opts ...grpc.CallOption) (*protomfx.GroupID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getGroupIDByThingID(ctx, groupIDByThingIDReq{thingID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	tg := res.(groupIDByThingIDRes)
	return &protomfx.GroupID{Value: tg.groupID}, nil
}

func encodeGetPubConfByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(pubConfByKeyReq)
	return &protomfx.PubConfByKeyReq{Key: req.key}, nil
}

func encodeGetConfigByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(configByThingIDReq)
	return &protomfx.ThingID{Value: req.thingID}, nil
}

func encodeUserAccessThingRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(userAccessThingReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeUserAccessProfileRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(userAccessProfileReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeUserAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(userAccessGroupReq)
	return &protomfx.UserAccessReq{Token: req.token, Id: req.id, Action: req.action}, nil
}

func encodeThingAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(thingAccessGroupReq)
	return &protomfx.ThingAccessReq{Key: req.key, Id: req.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &protomfx.Token{Value: req.key}, nil
}

func encodeGetGroupIDByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(groupIDByThingIDReq)
	return &protomfx.ThingID{Value: req.thingID}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.ThingID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeGetPubConfByKeyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.PubConfByKeyRes)
	return pubConfByKeyRes{publisherID: res.PublisherID, profileConfig: res.ProfileConfig}, nil
}

func decodeGetConfigByThingIDResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.ConfigByThingIDRes)
	return configByThingIDRes{config: res.GetConfig()}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}

func decodeGetGroupIDByThingIDResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*protomfx.GroupID)
	return groupIDByThingIDRes{groupID: res.GetValue()}, nil
}
