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
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

const (
	svcName = "mainflux.AuthService"
)

var _ mainflux.AuthServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	issue      endpoint.Endpoint
	identify   endpoint.Endpoint
	authorize  endpoint.Endpoint
	addPolicy  endpoint.Endpoint
	assign     endpoint.Endpoint
	members    endpoint.Endpoint
	assignRole endpoint.Endpoint
	timeout    time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(tracer opentracing.Tracer, conn *grpc.ClientConn, timeout time.Duration) mainflux.AuthServiceClient {
	return &grpcClient{
		issue: kitot.TraceClient(tracer, "issue")(kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			mainflux.UserIdentity{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			mainflux.UserIdentity{},
		).Endpoint()),
		authorize: kitot.TraceClient(tracer, "authorize")(kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		addPolicy: kitot.TraceClient(tracer, "add_policy")(kitgrpc.NewClient(
			conn,
			svcName,
			"AddPolicy",
			encodeAddPolicyRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),
		assign: kitot.TraceClient(tracer, "assign")(kitgrpc.NewClient(
			conn,
			svcName,
			"Assign",
			encodeAssignRequest,
			decodeAssignResponse,
			mainflux.AuthorizeRes{},
		).Endpoint()),
		members: kitot.TraceClient(tracer, "members")(kitgrpc.NewClient(
			conn,
			svcName,
			"Members",
			encodeMembersRequest,
			decodeMembersResponse,
			mainflux.MembersRes{},
		).Endpoint()),
		assignRole: kitot.TraceClient(tracer, "assign_role")(kitgrpc.NewClient(
			conn,
			svcName,
			"AssignRole",
			encodeAssignRoleRequest,
			decodeEmptyResponse,
			empty.Empty{},
		).Endpoint()),

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *mainflux.IssueReq, _ ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.issue(ctx, issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.Type})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.Token{Value: ir.id}, nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &mainflux.IssueReq{Id: req.id, Email: req.email, Type: req.keyType}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail()}, nil
}

func (client grpcClient) Identify(ctx context.Context, token *mainflux.Token, _ ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.UserIdentity{Id: ir.id, Email: ir.email}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &mainflux.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail()}, nil
}

func (client grpcClient) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.authorize(ctx, authReq{Token: req.GetToken(), Object: req.GetObject(), Subject: req.GetSubject(), Action: req.GetAction()})
	if err != nil {
		return &empty.Empty{}, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &mainflux.AuthorizeReq{
		Token:   req.Token,
		Object:  req.Object,
		Subject: req.Subject,
		Action:  req.Action,
	}, nil
}

func (client grpcClient) AddPolicy(ctx context.Context, req *mainflux.PolicyReq, opts ...grpc.CallOption) (r *empty.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.addPolicy(ctx, policyReq{Token: req.GetToken(), Object: req.GetObject(), Policy: req.GetPolicy()})
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func encodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &mainflux.PolicyReq{
		Token:  req.Token,
		Object: req.Object,
		Policy: req.Policy,
	}, nil
}

func (client grpcClient) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.members(ctx, membersReq{
		token:      req.GetToken(),
		groupID:    req.GetGroupID(),
		memberType: req.GetType(),
		offset:     req.GetOffset(),
		limit:      req.GetLimit(),
	})
	if err != nil {
		return &mainflux.MembersRes{}, err
	}

	mr := res.(orgMembersRes)

	return &mainflux.MembersRes{
		Offset:  mr.offset,
		Limit:   mr.limit,
		Total:   mr.total,
		Members: mr.orgMemberIDs,
	}, err
}

func (client grpcClient) AssignRole(ctx context.Context, req *mainflux.AssignRoleReq, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.assignRole(ctx, assignRoleReq{ID: req.GetId(), Role: req.GetRole()})
	if err != nil {
		return &empty.Empty{}, err
	}

	er := res.(emptyRes)
	return &empty.Empty{}, er.err
}

func encodeAssignRoleRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(assignRoleReq)
	return &mainflux.AssignRoleReq{
		Id:   req.ID,
		Role: req.Role,
	}, nil
}

func encodeMembersRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(membersReq)
	return &mainflux.MembersReq{
		Token:   req.token,
		Offset:  req.offset,
		Limit:   req.limit,
		GroupID: req.groupID,
		Type:    req.memberType,
	}, nil
}

func decodeMembersResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.MembersRes)
	return orgMembersRes{
		offset:       res.Offset,
		limit:        res.Limit,
		total:        res.Total,
		orgMemberIDs: res.Members,
	}, nil
}

func (client grpcClient) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	_, err = client.assign(ctx, assignReq{token: req.GetToken(), groupID: req.GetGroupID(), memberID: req.GetMemberID()})
	if err != nil {
		return &empty.Empty{}, err
	}

	return &empty.Empty{}, err
}

func encodeAssignRequest(_ context.Context, grpcRes interface{}) (interface{}, error) {
	req := grpcRes.(assignReq)
	return &mainflux.Assignment{
		Token:    req.token,
		GroupID:  req.groupID,
		MemberID: req.memberID,
	}, nil
}

func decodeAssignResponse(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &mainflux.AuthorizeReq{
		Token: req.Token,
	}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}
