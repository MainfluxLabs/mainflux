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
	issue                               kitgrpc.Handler
	refresh                             kitgrpc.Handler
	revokeSessions                      kitgrpc.Handler
	identify                            kitgrpc.Handler
	identifyRecovery                    kitgrpc.Handler
	authorize                           kitgrpc.Handler
	getOwnerIDByOrg                     kitgrpc.Handler
	assignRole                          kitgrpc.Handler
	retrieveRole                        kitgrpc.Handler
	retrieveRoles                       kitgrpc.Handler
	unassignRole                        kitgrpc.Handler
	createDormantOrgInvite              kitgrpc.Handler
	activateOrgInvite                   kitgrpc.Handler
	getDormantOrgInviteByPlatformInvite kitgrpc.Handler
	viewOrg                             kitgrpc.Handler
	getAdminsByOrg                      kitgrpc.Handler
	getOrgIDsByAdmin                    kitgrpc.Handler
	getOrgMembership                    kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc auth.Service) protomfx.AuthServiceServer {
	return &grpcServer{
		issue: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "issue")(issueEndpoint(svc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		refresh: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "refresh")(refreshEndpoint(svc)),
			decodeRefreshRequest,
			encodeIssueResponse,
		),
		revokeSessions: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "revoke_sessions")(revokeSessionsEndpoint(svc)),
			decodeRevokeSessionsRequest,
			encodeEmptyResponse,
		),
		identify: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
		identifyRecovery: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "identify_recovery")(identifyRecoveryEndpoint(svc)),
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
		retrieveRoles: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "retrieve_roles")(retrieveRolesEndpoint(svc)),
			decodeRetrieveRolesRequest,
			encodeRetrieveRolesResponse,
		),
		unassignRole: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "unassign_role")(unassignRoleEndpoint(svc)),
			decodeUnassignRoleRequest,
			encodeEmptyResponse,
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
		getDormantOrgInviteByPlatformInvite: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_dormant_org_invite_by_platform_invite")(getDormantOrgInviteByPlatformInviteEndpoint(svc)),
			decodeGetDormantOrgInviteByPlatformInviteRequest,
			encodeOrgInviteResponse,
		),
		viewOrg: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "view_org")(viewOrgEndpoint(svc)),
			decodeViewOrgRequest,
			encodeViewOrgResponse,
		),
		getAdminsByOrg: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_admins_by_org")(getAdminsByOrgEndpoint(svc)),
			decodeGetAdminsByOrgRequest,
			encodeGetAdminsByOrgResponse,
		),
		getOrgIDsByAdmin: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_org_ids_by_admin")(getOrgIDsByAdminEndpoint(svc)),
			decodeGetOrgIDsByAdminRequest,
			encodeGetOrgIDsByAdminResponse,
		),
		getOrgMembership: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "get_org_membership")(getOrgMembershipEndpoint(svc)),
			decodeGetOrgMembershipRequest,
			encodeGetOrgMembershipResponse,
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

func (s *grpcServer) Refresh(ctx context.Context, token *protomfx.Token) (*protomfx.Token, error) {
	_, res, err := s.refresh.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.Token), nil
}

func (s *grpcServer) RevokeSessions(ctx context.Context, req *protomfx.RevokeSessionsReq) (*emptypb.Empty, error) {
	_, res, err := s.revokeSessions.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*emptypb.Empty), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *protomfx.Token) (*protomfx.UserIdentity, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.UserIdentity), nil
}

func (s *grpcServer) IdentifyRecovery(ctx context.Context, token *protomfx.Token) (*protomfx.UserIdentity, error) {
	_, res, err := s.identifyRecovery.ServeGRPC(ctx, token)
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

func (s *grpcServer) RetrieveRoles(ctx context.Context, req *protomfx.RetrieveRolesReq) (*protomfx.RetrieveRolesRes, error) {
	_, res, err := s.retrieveRoles.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.RetrieveRolesRes), nil
}

func (s *grpcServer) UnassignRole(ctx context.Context, req *protomfx.UnassignRoleReq) (*emptypb.Empty, error) {
	_, res, err := s.unassignRole.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*emptypb.Empty), nil
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

func (s *grpcServer) GetDormantOrgInviteByPlatformInvite(ctx context.Context, req *protomfx.GetDormantOrgInviteByPlatformInviteReq) (*protomfx.OrgInvite, error) {
	_, res, err := s.getDormantOrgInviteByPlatformInvite.ServeGRPC(ctx, req)
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

func (s *grpcServer) GetAdminsByOrg(ctx context.Context, req *protomfx.OrgID) (*protomfx.OrgMemberships, error) {
	_, res, err := s.getAdminsByOrg.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.OrgMemberships), nil
}

func (s *grpcServer) GetOrgIDsByAdmin(ctx context.Context, req *protomfx.MemberID) (*protomfx.OrgIDs, error) {
	_, res, err := s.getOrgIDsByAdmin.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.OrgIDs), nil
}

func (s *grpcServer) GetOrgMembership(ctx context.Context, req *protomfx.OrgMembershipReq) (*protomfx.OrgMembership, error) {
	_, res, err := s.getOrgMembership.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*protomfx.OrgMembership), nil
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

func decodeRetrieveRolesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.RetrieveRolesReq)
	return retrieveRolesReq{ids: req.GetIds()}, nil
}

func encodeRetrieveRolesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveRolesRes)

	var roles []*protomfx.UserRole
	for userID, role := range res.roles {
		roles = append(roles, &protomfx.UserRole{
			UserId: userID,
			Role:   role,
		})
	}

	return &protomfx.RetrieveRolesRes{Roles: roles}, nil
}

func decodeUnassignRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.UnassignRoleReq)
	return unassignRoleReq{id: req.GetId()}, nil
}

func decodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.IssueReq)
	return issueReq{
		id:                req.GetId(),
		email:             req.GetEmail(),
		keyType:           req.GetType(),
		impersonatorID:    req.GetImpersonatorId(),
		impersonatorEmail: req.GetImpersonatorEmail(),
	}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(issueRes)
	return &protomfx.Token{Value: res.value}, nil
}

func decodeRefreshRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.Token)
	return refreshReq{token: req.GetValue()}, nil
}

func decodeRevokeSessionsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.RevokeSessionsReq)
	return revokeSessionsReq{id: req.GetId()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(identityRes)
	return &protomfx.UserIdentity{
		Id:                res.id,
		Email:             res.email,
		ImpersonatorId:    res.impersonatorID,
		ImpersonatorEmail: res.impersonatorEmail,
	}, nil
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

func decodeGetAdminsByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.OrgID)
	return adminsByOrgReq{orgID: req.GetValue()}, nil
}

func encodeGetAdminsByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(adminsByOrgRes)
	oms := make([]*protomfx.OrgMembership, 0, len(res.memberships))
	for _, om := range res.memberships {
		oms = append(oms, &protomfx.OrgMembership{
			MemberId: om.MemberID,
			Role:     om.Role,
		})
	}

	return &protomfx.OrgMemberships{Memberships: oms}, nil
}

func decodeGetOrgIDsByAdminRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.MemberID)
	return orgIDsByAdminReq{memberID: req.GetValue()}, nil
}

func encodeGetOrgIDsByAdminResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgIDsByAdminRes)
	return &protomfx.OrgIDs{Ids: res.orgIDs}, nil
}

func decodeGetOrgMembershipRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.OrgMembershipReq)
	return orgMembershipReq{orgID: req.GetOrgId(), memberID: req.GetMemberId()}, nil
}

func encodeGetOrgMembershipResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgMembershipRes)
	return &protomfx.OrgMembership{MemberId: res.memberID, Role: res.role}, nil
}

func decodeCreateDormantOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.CreateDormantOrgInviteReq)

	gis := []auth.GroupInvite{}

	for _, gi := range req.GetGroupInvites() {
		gis = append(gis, auth.GroupInvite{
			GroupID:    gi.GroupId,
			MemberRole: gi.MemberRole,
		})
	}

	return createDormantOrgInviteReq{
		token:            req.GetToken(),
		orgID:            req.GetOrgId(),
		inviteeRole:      req.GetInviteeRole(),
		groupInvites:     gis,
		platformInviteID: req.GetPlatformInviteId(),
	}, nil
}

func decodeGetDormantOrgInviteByPlatformInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.GetDormantOrgInviteByPlatformInviteReq)
	return getDormantOrgInviteByPlatformInviteReq{platformInviteID: req.GetPlatformInviteId()}, nil
}

func encodeOrgInviteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgInviteRes)
	groupInvites := make([]*protomfx.GroupInvite, 0, len(res.GroupInvites))
	for _, groupInvite := range res.GroupInvites {
		groupInvites = append(groupInvites, &protomfx.GroupInvite{
			GroupId:    groupInvite.GroupID,
			MemberRole: groupInvite.MemberRole,
		})
	}
	return &protomfx.OrgInvite{
		Id:           res.ID,
		OrgId:        res.OrgID,
		OrgName:      res.OrgName,
		InviteeRole:  res.InviteeRole,
		GroupInvites: groupInvites,
	}, nil
}

func decodeActivateOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ActivateOrgInviteReq)
	return activateOrgInviteReq{
		platformInviteID: req.GetPlatformInviteId(),
		userID:           req.GetUserId(),
		redirectPath:     req.GetRedirectPath(),
	}, nil
}

func decodeViewOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ViewOrgReq)
	return viewOrgReq{id: req.GetOrgId(), token: req.GetToken()}, nil
}

func encodeViewOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(orgRes)
	return &protomfx.Org{
		Id:      res.id,
		OwnerId: res.ownerID,
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
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingOrgID,
		err == apiutil.ErrMissingMemberID,
		err == apiutil.ErrMissingUserID:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		errors.Contains(err, auth.ErrAPIKeyExpired),
		errors.Contains(err, auth.ErrSessionReuse),
		errors.Contains(err, auth.ErrSessionExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, dbutil.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Contains(err, dbutil.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
