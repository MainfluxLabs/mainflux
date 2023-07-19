// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	mainflux "github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	issue          kitgrpc.Handler
	identify       kitgrpc.Handler
	authorize      kitgrpc.Handler
	canAccessGroup kitgrpc.Handler
	assign         kitgrpc.Handler
	members        kitgrpc.Handler
	saveRole       kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc auth.Service) mainflux.AuthServiceServer {
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
		canAccessGroup: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "can_access_group")(accessGroupEndpoint(svc)),
			decodeAccessGroupRequest,
			encodeEmptyResponse,
		),
		assign: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "assign")(assignEndpoint(svc)),
			decodeAssignRequest,
			encodeEmptyResponse,
		),
		members: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "members")(membersEndpoint(svc)),
			decodeMembersRequest,
			encodeMembersResponse,
		),
		saveRole: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "save_role")(saveRoleEndpoint(svc)),
			decodeSaveRoleRequest,
			encodeEmptyResponse,
		),
	}
}

func (s *grpcServer) Issue(ctx context.Context, req *mainflux.IssueReq) (*mainflux.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.Token), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *mainflux.Token) (*mainflux.UserIdentity, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.UserIdentity), nil
}

func (s *grpcServer) Authorize(ctx context.Context, req *mainflux.AuthorizeReq) (*empty.Empty, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) CanAccessGroup(ctx context.Context, req *mainflux.AccessGroupReq) (*empty.Empty, error) {
	_, res, err := s.canAccessGroup.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) Assign(ctx context.Context, token *mainflux.Assignment) (*empty.Empty, error) {
	_, res, err := s.assign.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) Members(ctx context.Context, req *mainflux.MembersReq) (*mainflux.MembersRes, error) {
	_, res, err := s.members.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.MembersRes), nil
}

func (s *grpcServer) SaveRole(ctx context.Context, req *mainflux.SaveRoleReq) (*empty.Empty, error) {
	_, res, err := s.saveRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func decodeSaveRoleRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.SaveRoleReq)
	return saveRoleReq{ID: req.GetId(), Role: req.GetRole()}, nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.IssueReq)
	return issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.GetType()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(issueRes)
	return &mainflux.Token{Value: res.value}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &mainflux.UserIdentity{Id: res.id, Email: res.email}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AuthorizeReq)
	return authReq{Token: req.GetToken()}, nil
}

func decodeAssignRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return assignReq{token: req.GetValue()}, nil
}

func decodeAccessGroupRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AccessGroupReq)
	return accessGroupReq{Token: req.GetToken(), GroupID: req.GetGroupID()}, nil
}

func decodeMembersRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.MembersReq)
	return membersReq{
		token:      req.GetToken(),
		groupID:    req.GetGroupID(),
		memberType: req.GetType(),
		offset:     req.Offset,
		limit:      req.Limit,
	}, nil
}

func encodeMembersResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(membersRes)
	return &mainflux.MembersRes{
		Total:   res.total,
		Offset:  res.offset,
		Limit:   res.limit,
		Members: res.members,
	}, nil
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
