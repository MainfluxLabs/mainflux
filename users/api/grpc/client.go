// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

const svcName = "protomfx.UsersService"

var _ protomfx.UsersServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout          time.Duration
	getUsersByIDs    endpoint.Endpoint
	getUsersByEmails endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.UsersServiceClient {
	return &grpcClient{
		timeout: timeout,
		getUsersByIDs: kitot.TraceClient(tracer, "get_users_by_ids")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetUsersByIDs",
			encodeGetUsersByIDsRequest,
			decodeGetUsersResponse,
			protomfx.UsersRes{},
		).Endpoint()),
		getUsersByEmails: kitot.TraceClient(tracer, "get_users_by_emails")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetUsersByEmails",
			encodeGetUsersByEmailsRequest,
			decodeGetUsersResponse,
			protomfx.UsersRes{},
		).Endpoint()),
	}
}

func (client grpcClient) GetUsersByIDs(ctx context.Context, req *protomfx.UsersByIDsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	pm := toPageMetadata(req.PageMetadata)
	res, err := client.getUsersByIDs(ctx, getUsersByIDsReq{ids: req.GetIds(), pageMetadata: pm})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersRes)

	return &protomfx.UsersRes{Users: ir.users, PageMetadata: ir.pageMetadata}, nil

}

func (client grpcClient) GetUsersByEmails(ctx context.Context, req *protomfx.UsersByEmailsReq, _ ...grpc.CallOption) (*protomfx.UsersRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getUsersByEmails(ctx, getUsersByEmailsReq{emails: req.GetEmails()})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersRes)

	return &protomfx.UsersRes{Users: ir.users}, nil

}

func encodeGetUsersByIDsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(getUsersByIDsReq)
	pm := toProtoPageMetadata(req.pageMetadata)

	return &protomfx.UsersByIDsReq{Ids: req.ids, PageMetadata: &pm}, nil
}

func encodeGetUsersByEmailsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(getUsersByEmailsReq)
	return &protomfx.UsersByEmailsReq{Emails: req.emails}, nil
}

func decodeGetUsersResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.UsersRes)
	return getUsersRes{users: res.GetUsers(), pageMetadata: res.GetPageMetadata()}, nil
}

func toPageMetadata(pm *protomfx.PageMetadata) users.PageMetadata {
	if pm == nil {
		return users.PageMetadata{}
	}
	return users.PageMetadata{
		Total:  pm.GetTotal(),
		Offset: pm.GetOffset(),
		Limit:  pm.GetLimit(),
		Email:  pm.GetEmail(),
		Order:  pm.GetOrder(),
		Dir:    pm.GetDir(),
	}
}

func toProtoPageMetadata(pm users.PageMetadata) protomfx.PageMetadata {
	return protomfx.PageMetadata{
		Total:  pm.Total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Email:  pm.Email,
		Order:  pm.Order,
		Dir:    pm.Dir,
	}
}
