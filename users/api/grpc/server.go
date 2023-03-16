// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.UsersServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	getUsersByIDs kitgrpc.Handler
}

// NewServer returns new UsersServiceServer instance.

func NewServer(tracer opentracing.Tracer, svc users.Service) mainflux.UsersServiceServer {
	return &grpcServer{
		getUsersByIDs: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_users_by_ids")(listUsersByIDsEndpoint(svc)),
			decodeGetUsersByIDsRequest,
			encodeGetUsersByIDsResponse,
		),
	}
}

func (s *grpcServer) GetUsersByIDs(ctx context.Context, req *mainflux.UsersReq) (*mainflux.UsersRes, error) {
	_, res, err := s.getUsersByIDs.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.UsersRes), nil
}

func decodeGetUsersByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.UsersReq)
	return getUsersByIDsReq{ids: req.GetIds()}, nil
}

func encodeGetUsersByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(getUsersByIDsRes)
	return &mainflux.UsersRes{Users: res.users}, nil
}

func encodeError(err error) error {
	switch err {
	case nil:
		return nil
	case errors.ErrMalformedEntity, apiutil.ErrMissingID:
		return status.Error(codes.InvalidArgument, "received invalid can access request")
	case errors.ErrAuthentication:
		return status.Error(codes.Unauthenticated, "missing or invalid credentials provided")
	case errors.ErrNotFound:
		return status.Error(codes.NotFound, "entity does not exist")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
