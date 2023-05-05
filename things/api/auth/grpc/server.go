// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	canAccessByKey kitgrpc.Handler
	canAccessByID  kitgrpc.Handler
	isChannelOwner kitgrpc.Handler
	identify       kitgrpc.Handler
	getGroupsByIDs kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc things.Service) mainflux.ThingsServiceServer {
	return &grpcServer{
		canAccessByKey: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_access")(canAccessEndpoint(svc)),
			decodeCanAccessByKeyRequest,
			encodeIdentityResponse,
		),
		canAccessByID: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_access_by_id")(canAccessByIDEndpoint(svc)),
			decodeCanAccessByIDRequest,
			encodeEmptyResponse,
		),
		isChannelOwner: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "is_channel_owner")(isChannelOwnerEndpoint(svc)),
			decodeIsChannelOwnerRequest,
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
	}
}

func (gs *grpcServer) CanAccessByKey(ctx context.Context, req *mainflux.AccessByKeyReq) (*mainflux.ThingID, error) {
	_, res, err := gs.canAccessByKey.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.ThingID), nil
}

func (gs *grpcServer) CanAccessByID(ctx context.Context, req *mainflux.AccessByIDReq) (*empty.Empty, error) {
	_, res, err := gs.canAccessByID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) IsChannelOwner(ctx context.Context, req *mainflux.ChannelOwnerReq) (*empty.Empty, error) {
	_, res, err := gs.isChannelOwner.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*empty.Empty), nil
}

func (gs *grpcServer) Identify(ctx context.Context, req *mainflux.Token) (*mainflux.ThingID, error) {
	_, res, err := gs.identify.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.ThingID), nil
}

func (gs *grpcServer) GetGroupsByIDs(ctx context.Context, req *mainflux.GroupsReq) (*mainflux.GroupsRes, error) {
	_, res, err := gs.getGroupsByIDs.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.GroupsRes), nil
}

func decodeCanAccessByKeyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AccessByKeyReq)
	return accessByKeyReq{thingKey: req.GetToken(), chanID: req.GetChanID()}, nil
}

func decodeCanAccessByIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AccessByIDReq)
	return accessByIDReq{thingID: req.GetThingID(), chanID: req.GetChanID()}, nil
}

func decodeIsChannelOwnerRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.ChannelOwnerReq)
	return channelOwnerReq{owner: req.GetOwner(), chanID: req.GetChanID()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return identifyReq{key: req.GetValue()}, nil
}

func decodeGetGroupsByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.GroupsReq)
	return getGroupsByIDsReq{ids: req.GetIds()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &mainflux.ThingID{Value: res.id}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeGetGroupsByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(getGroupsByIDsRes)
	return &mainflux.GroupsRes{Groups: res.groups}, nil
}

func encodeError(err error) error {
	switch err {
	case nil:
		return nil
	case apiutil.ErrMalformedEntity,
		apiutil.ErrMissingID,
		apiutil.ErrBearerKey:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.ErrAuthentication:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.ErrAuthorization:
		return status.Error(codes.PermissionDenied, err.Error())
	case things.ErrEntityConnected:
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.ErrNotFound:
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
