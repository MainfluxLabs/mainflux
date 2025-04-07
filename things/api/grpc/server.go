// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	getPubConfByKey      kitgrpc.Handler
	getConfigByThingID   kitgrpc.Handler
	canUserAccessThing   kitgrpc.Handler
	canUserAccessProfile kitgrpc.Handler
	canUserAccessGroup   kitgrpc.Handler
	canThingAccessGroup  kitgrpc.Handler
	identify             kitgrpc.Handler
	getGroupIDByThingID  kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc things.Service) protomfx.ThingsServiceServer {
	return &grpcServer{
		getPubConfByKey: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_pub_conf_by_key")(getPubConfByKeyEndpoint(svc)),
			decodeGetPubConfByKeyRequest,
			encodeGetPubConfByKeyResponse,
		),
		getConfigByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_config_by_thing_id")(getConfigByThingIDEndpoint(svc)),
			decodeGetConfigByThingIDRequest,
			encodeGetConfigByThingIDResponse,
		),
		canUserAccessThing: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_thing")(canUserAccessThingEndpoint(svc)),
			decodeUserAccessThingRequest,
			encodeEmptyResponse,
		),
		canUserAccessProfile: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_profile")(canUserAccessProfileEndpoint(svc)),
			decodeUserAccessProfileRequest,
			encodeEmptyResponse,
		),
		canUserAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_user_access_group")(canUserAccessGroupEndpoint(svc)),
			decodeUserAccessGroupRequest,
			encodeEmptyResponse,
		),
		canThingAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_thing_access_group")(canThingAccessGroupEndpoint(svc)),
			decodeThingAccessGroupRequest,
			encodeEmptyResponse,
		),
		identify: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentityResponse,
		),
		getGroupIDByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_id_by_thing_id")(getGroupIDByThingIDEndpoint(svc)),
			decodeGetGroupIDByThingIDRequest,
			encodeGetGroupIDByThingIDResponse,
		),
	}
}

func (gs *grpcServer) GetPubConfByKey(ctx context.Context, req *protomfx.PubConfByKeyReq) (*protomfx.PubConfByKeyRes, error) {
	_, res, err := gs.getPubConfByKey.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.PubConfByKeyRes), nil
}

func (gs *grpcServer) GetConfigByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.ConfigByThingIDRes, error) {
	_, res, err := gs.getConfigByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.ConfigByThingIDRes), nil
}

func (gs *grpcServer) CanUserAccessThing(ctx context.Context, req *protomfx.UserAccessReq) (*empty.Empty, error) {
	_, res, err := gs.canUserAccessThing.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) CanUserAccessProfile(ctx context.Context, req *protomfx.UserAccessReq) (*empty.Empty, error) {
	_, res, err := gs.canUserAccessProfile.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) CanUserAccessGroup(ctx context.Context, req *protomfx.UserAccessReq) (*empty.Empty, error) {
	_, res, err := gs.canUserAccessGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) CanThingAccessGroup(ctx context.Context, req *protomfx.ThingAccessReq) (*empty.Empty, error) {
	_, res, err := gs.canThingAccessGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) Identify(ctx context.Context, req *protomfx.Token) (*protomfx.ThingID, error) {
	_, res, err := gs.identify.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.ThingID), nil
}

func (gs *grpcServer) GetGroupIDByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.GroupID, error) {
	_, res, err := gs.getGroupIDByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.GroupID), nil
}

func decodeGetPubConfByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.PubConfByKeyReq)
	return pubConfByKeyReq{key: req.GetKey()}, nil
}

func decodeGetConfigByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ThingID)
	return configByThingIDReq{thingID: req.GetValue()}, nil
}

func decodeUserAccessThingRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessThingReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeUserAccessProfileRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessProfileReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeUserAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.UserAccessReq)
	return userAccessGroupReq{accessReq: accessReq{token: req.GetToken(), action: req.GetAction()}, id: req.GetId()}, nil
}

func decodeThingAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ThingAccessReq)
	return thingAccessGroupReq{key: req.GetKey(), id: req.GetId()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.Token)
	return identifyReq{key: req.GetValue()}, nil
}

func decodeGetGroupIDByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ThingID)
	return groupIDByThingIDReq{thingID: req.GetValue()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &protomfx.ThingID{Value: res.id}, nil
}

func encodeGetPubConfByKeyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(pubConfByKeyRes)
	return &protomfx.PubConfByKeyRes{PublisherID: res.publisherID, ProfileConfig: res.profileConfig}, nil
}

func encodeGetConfigByThingIDResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(configByThingIDRes)
	return &protomfx.ConfigByThingIDRes{Config: res.config}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeGetGroupIDByThingIDResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(groupIDByThingIDRes)
	return &protomfx.GroupID{Value: res.groupID}, nil
}

func encodeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingThingID,
		err == apiutil.ErrMissingProfileID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrInvalidAction,
		err == apiutil.ErrBearerToken,
		err == apiutil.ErrBearerKey:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, errors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
