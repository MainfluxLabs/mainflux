// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
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

var _ protomfx.AuthServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	issue                            endpoint.Endpoint
	identify                         endpoint.Endpoint
	authorize                        endpoint.Endpoint
	getOwnerIDByOrg                  endpoint.Endpoint
	retrieveRole                     endpoint.Endpoint
	assignRole                       endpoint.Endpoint
	createDormantOrgInvite           endpoint.Endpoint
	activateOrgInvite                endpoint.Endpoint
	getDormantInviteByPlatformInvite endpoint.Endpoint
	viewOrg                          endpoint.Endpoint
	timeout                          time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.AuthServiceClient {
	return &grpcClient{
		issue: kitot.TraceClient(tracer, "issue")(kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			protomfx.UserIdentity{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
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
		getDormantInviteByPlatformInvite: kitot.TraceClient(tracer, "get_dormant_invite_by_platform_invite")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetDormantInviteByPlatformInvite",
			encodeGetDormantInviteByPlatformInviteRequest,
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

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *protomfx.IssueReq, _ ...grpc.CallOption) (*protomfx.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.issue(ctx, issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.Type})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &protomfx.Token{Value: ir.id}, nil
}

func encodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(issueReq)
	return &protomfx.IssueReq{Id: req.id, Email: req.email, Type: req.keyType}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail()}, nil
}

func (client grpcClient) Identify(ctx context.Context, token *protomfx.Token, _ ...grpc.CallOption) (*protomfx.UserIdentity, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &protomfx.UserIdentity{Id: ir.id, Email: ir.email}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(identityReq)
	return &protomfx.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail()}, nil
}

func (client grpcClient) Authorize(ctx context.Context, req *protomfx.AuthorizeReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.authorize(ctx, authReq{Token: req.GetToken(), Object: req.GetObject(), Subject: req.GetSubject(), Action: req.GetAction()})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
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

func (client grpcClient) GetOwnerIDByOrg(ctx context.Context, req *protomfx.OrgID, _ ...grpc.CallOption) (*protomfx.OwnerID, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getOwnerIDByOrg(ctx, ownerIDByOrgReq{orgID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	oid := res.(ownerIDByOrgRes)
	return &protomfx.OwnerID{Value: oid.ownerID}, nil
}

func encodeGetOwnerIDByOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(ownerIDByOrgReq)
	return &protomfx.OrgID{Value: req.orgID}, nil
}

func decodeGetOwnerIDByOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OwnerID)
	return ownerIDByOrgRes{ownerID: res.GetValue()}, nil
}

func (client grpcClient) AssignRole(ctx context.Context, req *protomfx.AssignRoleReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.assignRole(ctx, assignRoleReq{ID: req.GetId(), Role: req.GetRole()})
	if err != nil {
		return &emptypb.Empty{}, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func encodeAssignRoleRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(assignRoleReq)
	return &protomfx.AssignRoleReq{
		Id:   req.ID,
		Role: req.Role,
	}, nil
}

func (client grpcClient) RetrieveRole(ctx context.Context, req *protomfx.RetrieveRoleReq, _ ...grpc.CallOption) (r *protomfx.RetrieveRoleRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.retrieveRole(ctx, retrieveRoleReq{id: req.GetId()})
	if err != nil {
		return &protomfx.RetrieveRoleRes{}, err
	}

	rr := res.(retrieveRoleRes)
	return &protomfx.RetrieveRoleRes{Role: rr.role}, err
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

func (client grpcClient) CreateDormantOrgInvite(ctx context.Context, req *protomfx.CreateDormantOrgInviteReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	gis := []auth.GroupInvite{}
	for _, gi := range req.GetGroupInvites() {
		gis = append(gis, auth.GroupInvite{
			GroupID:    gi.GroupID,
			MemberRole: gi.MemberRole,
		})
	}

	res, err := client.createDormantOrgInvite(ctx, createDormantOrgInviteReq{
		token:            req.GetToken(),
		orgID:            req.GetOrgID(),
		inviteeRole:      req.GetInviteeRole(),
		groupInvites:     gis,
		platformInviteID: req.GetPlatformInviteID(),
	})

	if err != nil {
		return &emptypb.Empty{}, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func encodeCreateDormantOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(createDormantOrgInviteReq)

	gis := make([]*protomfx.GroupInvite, 0, len(req.groupInvites))
	for _, gi := range req.groupInvites {
		gis = append(gis, &protomfx.GroupInvite{
			GroupID:    gi.GroupID,
			MemberRole: gi.MemberRole,
		})
	}

	return &protomfx.CreateDormantOrgInviteReq{
		Token:            req.token,
		OrgID:            req.orgID,
		InviteeRole:      req.inviteeRole,
		GroupInvites:     gis,
		PlatformInviteID: req.platformInviteID,
	}, nil
}

func (client grpcClient) ActivateOrgInvite(ctx context.Context, req *protomfx.ActivateOrgInviteReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.activateOrgInvite(ctx, activateOrgInviteReq{
		platformInviteID: req.GetPlatformInviteID(),
		userID:           req.GetUserID(),
		redirectPath:     req.GetRedirectPath(),
	})

	if err != nil {
		return &emptypb.Empty{}, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func encodeActivateOrgInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(activateOrgInviteReq)
	return &protomfx.ActivateOrgInviteReq{
		PlatformInviteID: req.platformInviteID,
		UserID:           req.userID,
		RedirectPath:     req.redirectPath,
	}, nil
}

func (client grpcClient) GetDormantInviteByPlatformInvite(ctx context.Context, req *protomfx.GetDormantInviteByPlatformInviteReq, _ ...grpc.CallOption) (*protomfx.OrgInvite, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getDormantInviteByPlatformInvite(ctx, getDormantInviteByPlatformInviteReq{
		platformInviteID: req.GetPlatformInviteID(),
	})
	if err != nil {
		return &protomfx.OrgInvite{}, err
	}

	orgInvite := res.(orgInviteRes)
	groupInvites := make([]*protomfx.GroupInvite, 0, len(orgInvite.invite.GroupInvites))
	for _, groupInvite := range orgInvite.invite.GroupInvites {
		groupInvites = append(groupInvites, &protomfx.GroupInvite{
			GroupID:    groupInvite.GroupID,
			MemberRole: groupInvite.MemberRole,
		})
	}

	return &protomfx.OrgInvite{
		Id:           orgInvite.invite.ID,
		OrgID:        orgInvite.invite.OrgID,
		OrgName:      orgInvite.invite.OrgName,
		InviteeRole:  orgInvite.invite.InviteeRole,
		GroupInvites: groupInvites,
	}, nil
}

func encodeGetDormantInviteByPlatformInviteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(getDormantInviteByPlatformInviteReq)
	return &protomfx.GetDormantInviteByPlatformInviteReq{PlatformInviteID: req.platformInviteID}, nil
}

func decodeOrgInviteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgInvite)

	groupInvites := make([]auth.GroupInvite, 0, len(res.GetGroupInvites()))
	for _, groupInvite := range res.GetGroupInvites() {
		groupInvites = append(groupInvites, auth.GroupInvite{
			GroupID:    groupInvite.GroupID,
			MemberRole: groupInvite.MemberRole,
		})
	}

	return orgInviteRes{
		invite: auth.OrgInvite{
			ID:           res.GetId(),
			OrgID:        res.GetOrgID(),
			OrgName:      res.GetOrgName(),
			InviteeRole:  res.GetInviteeRole(),
			GroupInvites: groupInvites,
		},
	}, nil
}

func (client grpcClient) ViewOrg(ctx context.Context, req *protomfx.ViewOrgReq, _ ...grpc.CallOption) (*protomfx.Org, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.viewOrg(ctx, viewOrgReq{id: req.GetOrgID(), token: req.GetToken()})
	if err != nil {
		return &protomfx.Org{}, err
	}

	or := res.(orgRes)
	return &protomfx.Org{
		Id:      or.id,
		OwnerID: or.ownerID,
		Name:    or.name,
	}, nil
}

func encodeViewOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(viewOrgReq)
	return &protomfx.ViewOrgReq{
		Token: req.token,
		OrgID: req.id,
	}, nil
}

func decodeOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.Org)
	return orgRes{
		id:      res.GetId(),
		ownerID: res.GetOwnerID(),
		name:    res.GetName(),
	}, nil
}

func decodeEmptyResponse(_ context.Context, _ any) (any, error) {
	return emptyRes{}, nil
}
