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
	authorize           kitgrpc.Handler
	identify            kitgrpc.Handler
	getGroupsByIDs      kitgrpc.Handler
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
		authorize: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "authorize")(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
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
		getGroupIDByThingID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_group_id_by_thing_id")(getGroupIDByThingIDEndpoint(svc)),
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

func (gs *grpcServer) Authorize(ctx context.Context, req *protomfx.AuthorizeReq) (*empty.Empty, error) {
	_, res, err := gs.authorize.ServeGRPC(ctx, req)
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

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.AuthorizeReq)
	return authorizeReq{token: req.GetToken(), object: req.GetObject(), subject: req.GetSubject(), action: req.GetAction()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.Token)
	return identifyReq{key: req.GetValue()}, nil
}

func decodeGetGroupsByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.GroupsReq)
	return getGroupsByIDsReq{ids: req.GetIds()}, nil
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
	return &protomfx.ConnByKeyRes{ProfileID: res.profileID, ThingID: res.thingID, ProfileConfig: res.config}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeGetGroupsByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(getGroupsByIDsRes)
	return &protomfx.GroupsRes{Groups: res.groups}, nil
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
