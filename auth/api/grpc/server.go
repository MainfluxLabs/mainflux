// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	issue                            kitgrpc.Handler
	identify                         kitgrpc.Handler
	authorize                        kitgrpc.Handler
	getOwnerIDByOrg                  kitgrpc.Handler
	assignRole                       kitgrpc.Handler
	retrieveRole                     kitgrpc.Handler
	createDormantOrgInvite           kitgrpc.Handler
	activateOrgInvite                kitgrpc.Handler
	getDormantInviteByPlatformInvite kitgrpc.Handler
	viewOrg                          kitgrpc.Handler
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
		getOwnerIDByOrg: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_owner_id_by_org")(getOwnerIDByOrgEndpoint(svc)),
			decodeGetOwnerIDByOrgRequest,
			encodeGetOwnerIDByOrgResponse,
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
		createDormantOrgInvite: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "create_dormant_org_invite")(createDormantOrgInviteEndpoint(svc)),
			decodeCreateDormantOrgInviteRequest,
			encodeEmptyResponse,
		),
		activateOrgInvite: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "activate_org_invite")(activateOrgInviteEndpoint(svc)),
			decodeActivateOrgInviteRequest,
			encodeEmptyResponse,
		),
		getDormantInviteByPlatformInvite: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_dormant_invite_by_platform_invite")(getDormantInviteByPlatformInviteEndpoint(svc)),
			decodeGetDormantInviteByPlatformInviteRequest,
			encodeOrgInviteResponse,
		),
		viewOrg: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "view_org")(viewOrgEndpoint(svc)),
			decodeViewOrgRequest,
			encodeViewOrgResponse,
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

func (s *grpcServer) Authorize(ctx context.Context, req *protomfx.AuthorizeReq) (*emptypb.Empty, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*emptypb.Empty), nil
}

func (s *grpcServer) GetOwnerIDByOrg(ctx context.Context, req *protomfx.OrgID) (*protomfx.OwnerID, error) {
	_, res, err := s.getOwnerIDByOrg.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.OwnerID), nil
}

func (s *grpcServer) AssignRole(ctx context.Context, req *protomfx.AssignRoleReq) (*emptypb.Empty, error) {
	_, res, err := s.assignRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*emptypb.Empty), nil
}

func (s *grpcServer) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq) (*protomfx.RetrieveRoleRes, error) {
	_, res, err := s.retrieveRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.RetrieveRoleRes), nil
}

func (s *grpcServer) CreateDormantOrgInvite(ctx context.Context, req *protomfx.CreateDormantOrgInviteReq) (*emptypb.Empty, error) {
	_, res, err := s.createDormantOrgInvite.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (s *grpcServer) ActivateOrgInvite(ctx context.Context, req *protomfx.ActivateOrgInviteReq) (*emptypb.Empty, error) {
	_, res, err := s.activateOrgInvite.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*emptypb.Empty), nil
}

func (s *grpcServer) GetDormantInviteByPlatformInvite(ctx context.Context, req *protomfx.GetDormantInviteByPlatformInviteReq) (*protomfx.OrgInvite, error) {
	_, res, err := s.getDormantInviteByPlatformInvite.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.OrgInvite), nil
}

func (s *grpcServer) ViewOrg(ctx context.Context, req *protomfx.ViewOrgReq) (*protomfx.Org, error) {
	_, res, err := s.viewOrg.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.Org), nil
}

func decodeAssignRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.AssignRoleReq)
	return assignRoleReq{ID: req.GetId(), Role: req.GetRole()}, nil
}

func decodeRetrieveRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.RetrieveRoleReq)
	return retrieveRoleReq{id: req.GetId()}, nil
}

func encodeRetrieveRoleResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveRoleRes)
	return &protomfx.RetrieveRoleRes{Role: res.role}, nil
}

func decodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.IssueReq)
	return issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.GetType()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(issueRes)
	return &protomfx.Token{Value: res.value}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(identityRes)
	return &protomfx.UserIdentity{Id: res.id, Email: res.email}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.AuthorizeReq)
	return authReq{Token: req.GetToken(), Object: req.GetObject(), Subject: req.GetSubject(), Action: req.GetAction()}, nil
}

func decodeGetOwnerIDByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.OrgID)
	return ownerIDByOrgReq{orgID: req.GetValue()}, nil
}

func encodeGetOwnerIDByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(ownerIDByOrgRes)
	return &protomfx.OwnerID{Value: res.ownerID}, nil
}

func decodeCreateDormantOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.CreateDormantOrgInviteReq)

	gis := []auth.GroupInvite{}

	for _, gi := range req.GetGroupInvites() {
		gis = append(gis, auth.GroupInvite{
			GroupID:    gi.GroupID,
			MemberRole: gi.MemberRole,
		})
	}

	return createDormantOrgInviteReq{
		token:            req.GetToken(),
		orgID:            req.GetOrgID(),
		inviteeRole:      req.GetInviteeRole(),
		groupInvites:     gis,
		platformInviteID: req.GetPlatformInviteID(),
	}, nil
}

func decodeGetDormantInviteByPlatformInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.GetDormantInviteByPlatformInviteReq)
	return getDormantInviteByPlatformInviteReq{platformInviteID: req.GetPlatformInviteID()}, nil
}

func encodeOrgInviteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgInviteRes)
	groupInvites := make([]*protomfx.GroupInvite, 0, len(res.invite.GroupInvites))
	for _, groupInvite := range res.invite.GroupInvites {
		groupInvites = append(groupInvites, &protomfx.GroupInvite{
			GroupID:    groupInvite.GroupID,
			MemberRole: groupInvite.MemberRole,
		})
	}
	return &protomfx.OrgInvite{
		Id:           res.invite.ID,
		OrgID:        res.invite.OrgID,
		OrgName:      res.invite.OrgName,
		InviteeRole:  res.invite.InviteeRole,
		GroupInvites: groupInvites,
	}, nil
}

func decodeActivateOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ActivateOrgInviteReq)
	return activateOrgInviteReq{
		platformInviteID: req.GetPlatformInviteID(),
		userID:           req.GetUserID(),
		redirectPath:     req.GetRedirectPath(),
	}, nil
}

func decodeViewOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ViewOrgReq)
	return viewOrgReq{id: req.GetOrgID(), token: req.GetToken()}, nil
}

func encodeViewOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgRes)
	return &protomfx.Org{
		Id:      res.id,
		OwnerID: res.ownerID,
		Name:    res.name,
	}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(emptyRes)
	return &emptypb.Empty{}, encodeError(res.err)
}

func encodeError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingOrgID,
		err == apiutil.ErrMissingUserID:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, dbutil.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
