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
	issue        endpoint.Endpoint
	identify     endpoint.Endpoint
	authorize    endpoint.Endpoint
	retrieveRole endpoint.Endpoint
	assignRole   endpoint.Endpoint
	timeout      time.Duration
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
		retrieveRole: kitot.TraceClient(tracer, "retrieve_role")(kitgrpc.NewClient(
			conn,
			svcName,
			"RetrieveRole",
			encodeRetrieveRoleRequest,
			decodeRetrieveRoleResponse,
			mainflux.RetrieveRoleRes{},
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

func (client grpcClient) RetrieveRole(ctx context.Context, req *mainflux.RetrieveRoleReq, _ ...grpc.CallOption) (r *mainflux.RetrieveRoleRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.retrieveRole(ctx, retrieveRoleReq{id: req.GetId()})
	if err != nil {
		return &mainflux.RetrieveRoleRes{}, err
	}

	rr := res.(retrieveRoleRes)
	return &mainflux.RetrieveRoleRes{Role: rr.role}, err
}

func encodeRetrieveRoleRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(retrieveRoleReq)
	return &mainflux.RetrieveRoleReq{
		Id: req.id,
	}, nil
}

func decodeRetrieveRoleResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.RetrieveRoleRes)
	return retrieveRoleRes{role: res.GetRole()}, nil
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
