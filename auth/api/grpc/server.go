// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	issue        kitgrpc.Handler
	identify     kitgrpc.Handler
	authorize    kitgrpc.Handler
	addPolicy    kitgrpc.Handler
	assign       kitgrpc.Handler
	members      kitgrpc.Handler
	assignRole   kitgrpc.Handler
	retrieveRole kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc auth.Service) protomfx.AuthServiceServer {
	return &grpcServer{
		issue: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "issue")(issueEndpoint(svc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		identify: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
		authorize: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "authorize")(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeEmptyResponse,
		),
		assignRole: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "assign_role")(assignRoleEndpoint(svc)),
			decodeAssignRoleRequest,
			encodeEmptyResponse,
		),
		retrieveRole: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "retrieve_role")(retrieveRoleEndpoint(svc)),
			decodeRetrieveRoleRequest,
			encodeRetrieveRoleResponse,
		),
	}
}

func (s *grpcServer) Issue(ctx context.Context, req *protomfx.IssueReq) (*protomfx.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.Token), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *protomfx.Token) (*protomfx.UserIdentity, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.UserIdentity), nil
}

func (s *grpcServer) Authorize(ctx context.Context, req *protomfx.AuthorizeReq) (*empty.Empty, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) AssignRole(ctx context.Context, req *protomfx.AssignRoleReq) (*empty.Empty, error) {
	_, res, err := s.assignRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq) (*protomfx.RetrieveRoleRes, error) {
	_, res, err := s.retrieveRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.RetrieveRoleRes), nil
}

func decodeAssignRoleRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.AssignRoleReq)
	return assignRoleReq{ID: req.GetId(), Role: req.GetRole()}, nil
}

func decodeRetrieveRoleRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.RetrieveRoleReq)
	return retrieveRoleReq{id: req.GetId()}, nil
}

func encodeRetrieveRoleResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(retrieveRoleRes)
	return &protomfx.RetrieveRoleRes{Role: res.role}, nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.IssueReq)
	return issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.GetType()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(issueRes)
	return &protomfx.Token{Value: res.value}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &protomfx.UserIdentity{Id: res.id, Email: res.email}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.AuthorizeReq)
	return authReq{Token: req.GetToken(), Object: req.GetObject(), Subject: req.Subject, Action: req.GetAction()}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
