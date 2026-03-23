// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	domainusers "github.com/MainfluxLabs/mainflux/pkg/domain/users"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

const svcName = "protomfx.UsersService"

var _ domainusers.Client = (*grpcClient)(nil)

type grpcClient struct {
	timeout          time.Duration
	getUsersByIDs    endpoint.Endpoint
	getUsersByEmails endpoint.Endpoint
}

// NewClient returns new gRPC client instance implementing domainusers.Client.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) domainusers.Client {
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

func (client grpcClient) GetUsersByIDs(ctx context.Context, ids []string, pm domainusers.PageMetadata) (domainusers.UsersPage, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getUsersByIDs(ctx, getUsersByIDsReq{ids: ids, pageMetadata: pm})
	if err != nil {
		return domainusers.UsersPage{}, err
	}

	ir := res.(getUsersRes)
	total := uint64(0)
	if ir.pageMetadata != nil {
		total = ir.pageMetadata.GetTotal()
	}
	return domainusers.UsersPage{
		Total: total,
		Users: protoUsersToDomain(ir.users),
	}, nil
}

func (client grpcClient) GetUsersByEmails(ctx context.Context, emails []string) ([]domainusers.User, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getUsersByEmails(ctx, getUsersByEmailsReq{emails: emails})
	if err != nil {
		return nil, err
	}

	ir := res.(getUsersRes)
	return protoUsersToDomain(ir.users), nil
}

func protoUsersToDomain(users []*protomfx.User) []domainusers.User {
	if users == nil {
		return nil
	}
	out := make([]domainusers.User, 0, len(users))
	for _, u := range users {
		out = append(out, domainusers.User{
			ID:     u.GetId(),
			Email:  u.GetEmail(),
			Status: u.GetStatus(),
		})
	}
	return out
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

func toProtoPageMetadata(pm domainusers.PageMetadata) protomfx.PageMetadata {
	return protomfx.PageMetadata{
		Total:  pm.Total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Email:  pm.Email,
		Order:  pm.Order,
		Dir:    pm.Dir,
	}
}
