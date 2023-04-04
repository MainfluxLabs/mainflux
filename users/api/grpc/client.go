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
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

const svcName = "mainflux.UsersService"

var _ mainflux.UsersServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout          time.Duration
	getUsersByIDs    endpoint.Endpoint
	getUsersByEmails endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) mainflux.UsersServiceClient {
	return &grpcClient{
		timeout: timeout,
		getUsersByIDs: kitot.TraceClient(tracer, "get_users_by_ids")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetUsersByIDs",
			encodeGetUsersByIDsRequest,
			decodeGetUsersResponse,
			mainflux.UsersRes{},
		).Endpoint()),
		getUsersByEmails: kitot.TraceClient(tracer, "get_users_by_emails")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetUsersByEmails",
			encodeGetUsersByEmailsRequest,
			decodeGetUsersResponse,
			mainflux.UsersRes{},
		).Endpoint()),
	}
}

func (clent grpcClient) GetUsersByIDs(ctx context.Context, req *mainflux.UsersByIDsReq, _ ...grpc.CallOption) (*mainflux.UsersRes, error) {
	ctx, close := context.WithTimeout(ctx, clent.timeout)
	defer close()

	res, err := clent.getUsersByIDs(ctx, getUsersByIDsReq{ids: req.GetIds()})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersRes)

	return &mainflux.UsersRes{Users: ir.users}, nil

}

func (clent grpcClient) GetUsersByEmails(ctx context.Context, req *mainflux.UsersByEmailsReq, _ ...grpc.CallOption) (*mainflux.UsersRes, error) {
	ctx, close := context.WithTimeout(ctx, clent.timeout)
	defer close()

	res, err := clent.getUsersByEmails(ctx, getUsersByEmailsReq{emails: req.GetEmails()})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersRes)

	return &mainflux.UsersRes{Users: ir.users}, nil

}

func encodeGetUsersByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(getUsersByIDsReq)
	return &mainflux.UsersByIDsReq{Ids: req.ids}, nil
}

func encodeGetUsersByEmailsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(getUsersByEmailsReq)
	return &mainflux.UsersByEmailsReq{Emails: req.emails}, nil
}

func decodeGetUsersResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UsersRes)
	return getUsersRes{users: res.GetUsers()}, nil
}
