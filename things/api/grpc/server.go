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
	getConnByKey        kitgrpc.Handler
	canAccessGroup      kitgrpc.Handler
	identify            kitgrpc.Handler
	getGroupsByIDs      kitgrpc.Handler
	getProfileByThingID kitgrpc.Handler
	getGroupIDByThingID kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc things.Service) protomfx.ThingsServiceServer {
	return &grpcServer{
		getConnByKey: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_conn_by_key")(getConnByKeyEndpoint(svc)),
			decodeGetConnByKeyRequest,
			encodeGetConnByKeyResponse,
		),
		canAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_access_group")(canAccessGroupEndpoint(svc)),
			decodeCanAccessGroupRequest,
			encodeEmptyResponse,
		),
		identify: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentityResponse,
		),
		getGroupsByIDs: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_groups_by_ids")(listGroupsByIDsEndpoint(svc)),
			decodeGetGroupsByIDsRequest,
			encodeGetGroupsByIDsResponse,
		),
		getProfileByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_profile_by_thing_id")(getProfileByThingIDEndpoint(svc)),
			decodeGetProfileByThingIDRequest,
			encodeGetProfileByThingIDResponse,
		),
		getGroupIDByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_id_by_thing_id")(GetGroupIDByThingIDEndpoint(svc)),
			decodeGetGroupIDByThingIDRequest,
			encodeGetGroupIDByThingIDResponse,
		),
	}
}

func (gs *grpcServer) GetConnByKey(ctx context.Context, req *protomfx.ConnByKeyReq) (*protomfx.ConnByKeyRes, error) {
	_, res, err := gs.getConnByKey.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.ConnByKeyRes), nil
}

func (gs *grpcServer) CanAccessGroup(ctx context.Context, req *protomfx.AccessGroupReq) (*empty.Empty, error) {
	_, res, err := gs.canAccessGroup.ServeGRPC(ctx, req)
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

func (gs *grpcServer) GetGroupsByIDs(ctx context.Context, req *protomfx.GroupsReq) (*protomfx.GroupsRes, error) {
	_, res, err := gs.getGroupsByIDs.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.GroupsRes), nil
}

func (gs *grpcServer) GetProfileByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.ProfileByThingIDRes, error) {
	_, res, err := gs.getProfileByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.ProfileByThingIDRes), nil
}

func (gs *grpcServer) GetGroupIDByThingID(ctx context.Context, req *protomfx.ThingID) (*protomfx.GroupID, error) {
	_, res, err := gs.getGroupIDByThingID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.GroupID), nil
}

func decodeGetConnByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ConnByKeyReq)
	return connByKeyReq{key: req.GetKey()}, nil
}

func decodeCanAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.AccessGroupReq)
	return accessGroupReq{token: req.GetToken(), groupID: req.GetGroupID(), action: req.GetAction(), object: req.GetObject(), subject: req.GetSubject()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.Token)
	return identifyReq{key: req.GetValue()}, nil
}

func decodeGetGroupsByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.GroupsReq)
	return getGroupsByIDsReq{ids: req.GetIds()}, nil
}

func decodeGetProfileByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ThingID)
	return profileByThingIDReq{thingID: req.GetValue()}, nil
}

func decodeGetGroupIDByThingIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.ThingID)
	return groupIDByThingIDReq{thingID: req.GetValue()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &protomfx.ThingID{Value: res.id}, nil
}

func encodeGetConnByKeyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(connByKeyRes)
	return &protomfx.ConnByKeyRes{ChannelID: res.channelID, ThingID: res.thingID, Profile: res.profile}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeGetGroupsByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(getGroupsByIDsRes)
	return &protomfx.GroupsRes{Groups: res.groups}, nil
}

func encodeGetProfileByThingIDResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(profileByThingIDRes)
	return &protomfx.ProfileByThingIDRes{Profile: res.profile}, nil
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
		err == apiutil.ErrMissingID,
		err == apiutil.ErrBearerKey:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, things.ErrEntityConnected):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, errors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
