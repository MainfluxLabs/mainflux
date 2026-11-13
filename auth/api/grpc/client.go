// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	svcName = "protomfx.AuthService"
)

var _ domain.AuthClient = (*grpcClient)(nil)

type grpcClient struct {
	issue                               endpoint.Endpoint
	refresh                             endpoint.Endpoint
	revokeSessions                      endpoint.Endpoint
	identify                            endpoint.Endpoint
	identifyRecovery                    endpoint.Endpoint
	authorize                           endpoint.Endpoint
	getOwnerIDByOrg                     endpoint.Endpoint
	retrieveRole                        endpoint.Endpoint
	assignRole                          endpoint.Endpoint
	createDormantOrgInvite              endpoint.Endpoint
	activateOrgInvite                   endpoint.Endpoint
	getDormantOrgInviteByPlatformInvite endpoint.Endpoint
	viewOrg                             endpoint.Endpoint
	getAdminsByOrg                      endpoint.Endpoint
	getOrgIDsByAdmin                    endpoint.Endpoint
	getOrgMembership                    endpoint.Endpoint
	timeout                             time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) domain.AuthClient {
	return &grpcClient{
		refresh: kitot.TraceClient(tracer, "refresh")(kitgrpc.NewClient(
			conn,
			svcName,
			"Refresh",
			encodeRefreshRequest,
			decodeIssueResponse,
			protomfx.Token{},
		).Endpoint()),
		revokeSessions: kitot.TraceClient(tracer, "revoke_sessions")(kitgrpc.NewClient(
			conn,
			svcName,
			"RevokeSessions",
			encodeRevokeSessionsRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		issue: kitot.TraceClient(tracer, "issue")(kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			protomfx.Token{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			protomfx.UserIdentity{},
		).Endpoint()),
		identifyRecovery: kitot.TraceClient(tracer, "identify_recovery")(kitgrpc.NewClient(
			conn,
			svcName,
			"IdentifyRecovery",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			protomfx.UserIdentity{},
		).Endpoint()),
		authorize: kitot.TraceClient(tracer, "authorize")(kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		getOwnerIDByOrg: kitot.TraceClient(tracer, "get_owner_id_by_org")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetOwnerIDByOrg",
			encodeGetOwnerIDByOrgRequest,
			decodeGetOwnerIDByOrgResponse,
			protomfx.OwnerID{},
		).Endpoint()),
		retrieveRole: kitot.TraceClient(tracer, "retrieve_role")(kitgrpc.NewClient(
			conn,
			svcName,
			"RetrieveRole",
			encodeRetrieveRoleRequest,
			decodeRetrieveRoleResponse,
			protomfx.RetrieveRoleRes{},
		).Endpoint()),
		assignRole: kitot.TraceClient(tracer, "assign_role")(kitgrpc.NewClient(
			conn,
			svcName,
			"AssignRole",
			encodeAssignRoleRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		createDormantOrgInvite: kitot.TraceClient(tracer, "create_dormant_org_invite")(kitgrpc.NewClient(
			conn,
			svcName,
			"CreateDormantOrgInvite",
			encodeCreateDormantOrgInviteRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		activateOrgInvite: kitot.TraceClient(tracer, "activate_org_invite")(kitgrpc.NewClient(
			conn,
			svcName,
			"ActivateOrgInvite",
			encodeActivateOrgInviteRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
		getDormantOrgInviteByPlatformInvite: kitot.TraceClient(tracer, "get_dormant_org_invite_by_platform_invite")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetDormantOrgInviteByPlatformInvite",
			encodeGetDormantOrgInviteByPlatformInviteRequest,
			decodeOrgInviteResponse,
			protomfx.OrgInvite{},
		).Endpoint()),
		viewOrg: kitot.TraceClient(tracer, "view_org")(kitgrpc.NewClient(
			conn,
			svcName,
			"ViewOrg",
			encodeViewOrgRequest,
			decodeOrgResponse,
			protomfx.Org{},
		).Endpoint()),
		getAdminsByOrg: kitot.TraceClient(tracer, "get_admins_by_org")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetAdminsByOrg",
			encodeGetAdminsByOrgRequest,
			decodeGetAdminsByOrgResponse,
			protomfx.OrgMemberships{},
		).Endpoint()),
		getOrgIDsByAdmin: kitot.TraceClient(tracer, "get_org_ids_by_admin")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetOrgIDsByAdmin",
			encodeGetOrgIDsByAdminRequest,
			decodeGetOrgIDsByAdminResponse,
			protomfx.OrgIDs{},
		).Endpoint()),
		getOrgMembership: kitot.TraceClient(tracer, "get_org_membership")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetOrgMembership",
			encodeGetOrgMembershipRequest,
			decodeGetOrgMembershipResponse,
			protomfx.OrgMembership{},
		).Endpoint()),

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, id, email string, keyType uint32) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{id: id, email: email, keyType: keyType})
	if err != nil {
		return "", err
	}

	ir := res.(issueRes)
	return ir.value, nil
}

func (client grpcClient) IssueImpersonated(ctx context.Context, id, email string, impersonator domain.Identity) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	req := issueReq{
		id:                id,
		email:             email,
		keyType:           domain.LoginKey,
		impersonatorID:    impersonator.ID,
		impersonatorEmail: impersonator.Email,
	}

	res, err := client.issue(ctx, req)
	if err != nil {
		return "", err
	}

	ir := res.(issueRes)
	return ir.value, nil
}

func (client grpcClient) RefreshToken(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.refresh(ctx, refreshReq{token: token})
	if err != nil {
		return "", err
	}

	ir := res.(issueRes)
	return ir.value, nil
}

func (client grpcClient) RevokeSessions(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.revokeSessions(ctx, revokeSessionsReq{id: id})
	if err != nil {
		return err
	}

	er := res.(emptyRes)
	return er.err
}

func encodeRefreshRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(refreshReq)
	return &protomfx.Token{Value: req.token}, nil
}

func encodeRevokeSessionsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(revokeSessionsReq)
	return &protomfx.RevokeSessionsReq{Id: req.id}, nil
}

func encodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(issueReq)
	return &protomfx.IssueReq{
		Id:                req.id,
		Email:             req.email,
		Type:              req.keyType,
		ImpersonatorId:    req.impersonatorID,
		ImpersonatorEmail: req.impersonatorEmail,
	}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.Token)
	return issueRes{value: res.GetValue()}, nil
}

func (client grpcClient) Identify(ctx context.Context, token string) (domain.Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identityReq{token: token})
	if err != nil {
		return domain.Identity{}, err
	}

	ir := res.(identityRes)
	identity := domain.Identity{
		ID:    ir.id,
		Email: ir.email,
	}

	if ir.impersonatorID != "" {
		identity.Impersonator = &domain.Identity{
			ID:    ir.impersonatorID,
			Email: ir.impersonatorEmail,
		}
	}

	return identity, nil
}

func (client grpcClient) IdentifyRecovery(ctx context.Context, token string) (domain.Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identifyRecovery(ctx, identityReq{token: token})
	if err != nil {
		return domain.Identity{}, err
	}

	ir := res.(identityRes)
	return domain.Identity{ID: ir.id, Email: ir.email}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(identityReq)
	return &protomfx.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.UserIdentity)
	ir := identityRes{
		id:                res.GetId(),
		email:             res.GetEmail(),
		impersonatorID:    res.GetImpersonatorId(),
		impersonatorEmail: res.GetImpersonatorEmail(),
	}

	return ir, nil
}

func (client grpcClient) Authorize(ctx context.Context, ar domain.AuthzReq) error {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authorize(ctx, authReq{Token: ar.Token, Object: ar.Object, Subject: ar.Subject, Action: ar.Action})
	if err != nil {
		return err
	}

	er := res.(emptyRes)
	return er.err
}

func encodeAuthorizeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(authReq)
	return &protomfx.AuthorizeReq{
		Token:   req.Token,
		Object:  req.Object,
		Subject: req.Subject,
		Action:  req.Action,
	}, nil
}

func (client grpcClient) GetOwnerIDByOrg(ctx context.Context, orgID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getOwnerIDByOrg(ctx, ownerIDByOrgReq{orgID: orgID})
	if err != nil {
		return "", err
	}

	oid := res.(ownerIDByOrgRes)
	return oid.ownerID, nil
}

func encodeGetOwnerIDByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(ownerIDByOrgReq)
	return &protomfx.OrgID{Value: req.orgID}, nil
}

func decodeGetOwnerIDByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OwnerID)
	return ownerIDByOrgRes{ownerID: res.GetValue()}, nil
}

func (client grpcClient) AssignRole(ctx context.Context, id, role string) error {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.assignRole(ctx, assignRoleReq{ID: id, Role: role})
	if err != nil {
		return err
	}

	er := res.(emptyRes)
	return er.err
}

func encodeAssignRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(assignRoleReq)
	return &protomfx.AssignRoleReq{
		Id:   req.ID,
		Role: req.Role,
	}, nil
}

func (client grpcClient) RetrieveRole(ctx context.Context, id string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveRole(ctx, retrieveRoleReq{id: id})
	if err != nil {
		return "", err
	}

	rr := res.(retrieveRoleRes)
	return rr.role, nil
}

func encodeRetrieveRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(retrieveRoleReq)

	return &protomfx.RetrieveRoleReq{
		Id: req.id,
	}, nil
}

func decodeRetrieveRoleResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.RetrieveRoleRes)
	return retrieveRoleRes{role: res.GetRole()}, nil
}

func (client grpcClient) CreateDormantOrgInvite(ctx context.Context, token, orgID, inviteeRole, platformInviteID string, groupInvites []domain.GroupInvite) error {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.createDormantOrgInvite(ctx, createDormantOrgInviteReq{
		token:            token,
		orgID:            orgID,
		inviteeRole:      inviteeRole,
		groupInvites:     groupInvites,
		platformInviteID: platformInviteID,
	})

	if err != nil {
		return err
	}

	er := res.(emptyRes)
	return er.err
}

func encodeCreateDormantOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(createDormantOrgInviteReq)

	gis := make([]*protomfx.GroupInvite, 0, len(req.groupInvites))
	for _, gi := range req.groupInvites {
		gis = append(gis, &protomfx.GroupInvite{
			GroupId:    gi.GroupID,
			MemberRole: gi.MemberRole,
		})
	}

	return &protomfx.CreateDormantOrgInviteReq{
		Token:            req.token,
		OrgId:            req.orgID,
		InviteeRole:      req.inviteeRole,
		GroupInvites:     gis,
		PlatformInviteId: req.platformInviteID,
	}, nil
}

func (client grpcClient) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, redirectPath string) error {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.activateOrgInvite(ctx, activateOrgInviteReq{
		platformInviteID: platformInviteID,
		userID:           userID,
		redirectPath:     redirectPath,
	})

	if err != nil {
		return err
	}

	er := res.(emptyRes)
	return er.err
}

func encodeActivateOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(activateOrgInviteReq)

	return &protomfx.ActivateOrgInviteReq{
		PlatformInviteId: req.platformInviteID,
		UserId:           req.userID,
		RedirectPath:     req.redirectPath,
	}, nil
}

func (client grpcClient) GetDormantOrgInviteByPlatformInvite(ctx context.Context, platformInviteID string) (domain.OrgInvite, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getDormantOrgInviteByPlatformInvite(ctx, getDormantOrgInviteByPlatformInviteReq{
		platformInviteID: platformInviteID,
	})

	if err != nil {
		return domain.OrgInvite{}, err
	}

	orgInvite := res.(orgInviteRes)

	return orgInvite.OrgInvite, nil
}

func encodeGetDormantOrgInviteByPlatformInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(getDormantOrgInviteByPlatformInviteReq)
	return &protomfx.GetDormantOrgInviteByPlatformInviteReq{PlatformInviteId: req.platformInviteID}, nil
}

func decodeOrgInviteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgInvite)

	groupInvites := make([]auth.GroupInvite, 0, len(res.GetGroupInvites()))
	for _, groupInvite := range res.GetGroupInvites() {
		groupInvites = append(groupInvites, auth.GroupInvite{
			GroupID:    groupInvite.GroupId,
			MemberRole: groupInvite.MemberRole,
		})
	}

	return orgInviteRes{
		auth.OrgInvite{
			ID:           res.GetId(),
			OrgID:        res.GetOrgId(),
			OrgName:      res.GetOrgName(),
			InviteeRole:  res.GetInviteeRole(),
			GroupInvites: groupInvites,
		},
	}, nil
}

func (client grpcClient) ViewOrg(ctx context.Context, token, orgID string) (domain.Org, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.viewOrg(ctx, viewOrgReq{id: orgID, token: token})
	if err != nil {
		return domain.Org{}, err
	}

	or := res.(orgRes)
	return domain.Org{
		ID:      or.id,
		OwnerID: or.ownerID,
		Name:    or.name,
	}, nil
}

func encodeViewOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(viewOrgReq)
	return &protomfx.ViewOrgReq{
		Token: req.token,
		OrgId: req.id,
	}, nil
}

func decodeOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.Org)
	return orgRes{
		id:      res.GetId(),
		ownerID: res.GetOwnerId(),
		name:    res.GetName(),
	}, nil
}

func (client grpcClient) GetAdminsByOrg(ctx context.Context, orgID string) ([]domain.OrgMembership, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getAdminsByOrg(ctx, adminsByOrgReq{orgID: orgID})
	if err != nil {
		return nil, err
	}

	ar := res.(adminsByOrgRes)
	return ar.memberships, nil
}

func encodeGetAdminsByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(adminsByOrgReq)
	return &protomfx.OrgID{Value: req.orgID}, nil
}

func decodeGetAdminsByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgMemberships)
	oms := make([]domain.OrgMembership, 0, len(res.GetMemberships()))
	for _, om := range res.GetMemberships() {
		oms = append(oms, domain.OrgMembership{
			MemberID: om.GetMemberId(),
			Role:     om.GetRole(),
		})
	}

	return adminsByOrgRes{memberships: oms}, nil
}

func (client grpcClient) GetOrgIDsByAdmin(ctx context.Context, memberID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getOrgIDsByAdmin(ctx, orgIDsByAdminReq{memberID: memberID})
	if err != nil {
		return nil, err
	}

	or := res.(orgIDsByAdminRes)
	return or.orgIDs, nil
}

func encodeGetOrgIDsByAdminRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(orgIDsByAdminReq)
	return &protomfx.MemberID{Value: req.memberID}, nil
}

func decodeGetOrgIDsByAdminResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgIDs)
	return orgIDsByAdminRes{orgIDs: res.GetIds()}, nil
}

func (client grpcClient) GetOrgMembership(ctx context.Context, orgID, memberID string) (domain.OrgMembership, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.getOrgMembership(ctx, orgMembershipReq{orgID: orgID, memberID: memberID})
	if err != nil {
		return domain.OrgMembership{}, err
	}

	mr := res.(orgMembershipRes)
	return domain.OrgMembership{OrgID: orgID, MemberID: mr.memberID, Role: mr.role}, nil
}

func encodeGetOrgMembershipRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(orgMembershipReq)
	return &protomfx.OrgMembershipReq{OrgId: req.orgID, MemberId: req.memberID}, nil
}

func decodeGetOrgMembershipResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgMembership)
	return orgMembershipRes{memberID: res.GetMemberId(), role: res.GetRole()}, nil
}

func decodeEmptyResponse(_ context.Context, _ any) (any, error) {
	return emptyRes{}, nil
}
