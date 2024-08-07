// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.UsersServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	getUsersByIDs    kitgrpc.Handler
	getUsersByEmails kitgrpc.Handler
}

// NewServer returns new UsersServiceServer instance.

func NewServer(tracer opentracing.Tracer, svc users.Service) protomfx.UsersServiceServer {
	return &grpcServer{
		getUsersByIDs: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_users_by_ids")(listUsersByIDsEndpoint(svc)),
			decodeGetUsersByIDsRequest,
			encodeGetUsersResponse,
		),
		getUsersByEmails: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_users_by_emails")(listUsersByEmailsEndpoint(svc)),
			decodeGetUsersByEmailsRequest,
			encodeGetUsersResponse,
		),
	}
}

func (s *grpcServer) GetUsersByIDs(ctx context.Context, req *protomfx.UsersByIDsReq) (*protomfx.UsersRes, error) {
	_, res, err := s.getUsersByIDs.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.UsersRes), nil
}

func (s *grpcServer) GetUsersByEmails(ctx context.Context, req *protomfx.UsersByEmailsReq) (*protomfx.UsersRes, error) {
	_, res, err := s.getUsersByEmails.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.UsersRes), nil
}

func decodeGetUsersByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.UsersByIDsReq)
	return getUsersByIDsReq{ids: req.GetIds()}, nil
}

func decodeGetUsersByEmailsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.UsersByEmailsReq)
	return getUsersByEmailsReq{emails: req.GetEmails()}, nil
}

func encodeGetUsersResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(getUsersRes)
	return &protomfx.UsersRes{Users: res.users}, nil
}

func encodeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingID:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
