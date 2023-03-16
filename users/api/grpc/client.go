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
	timeout       time.Duration
	getUsersByIDs endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) mainflux.UsersServiceClient {
	return &grpcClient{
		getUsersByIDs: kitot.TraceClient(tracer, "get_users_by_ids")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetUsersByIDs",
			encodeGetUsersByIDsRequest,
			decodeGetUsersByIDsResponse,
			mainflux.UsersRes{},
		).Endpoint()),
	}
}

func (clent grpcClient) GetUsersByIDs(ctx context.Context, req *mainflux.UsersReq, _ ...grpc.CallOption) (*mainflux.UsersRes, error) {
	ctx, close := context.WithTimeout(ctx, clent.timeout)
	defer close()

	res, err := clent.getUsersByIDs(ctx, getUsersByIDsReq{IDs: req.GetIds()})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersByIDsRes)

	return &mainflux.UsersRes{Users: ir.Users}, nil

}

func encodeGetUsersByIDsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(getUsersByIDsReq)
	return &getUsersByIDsReq{IDs: req.IDs}, nil
}

func decodeGetUsersByIDsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UsersRes)
	return getUsersByIDsRes{Users: res.GetUsers()}, nil
}
