// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

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
	issue                  endpoint.Endpoint
	identify               endpoint.Endpoint
	authorize              endpoint.Endpoint
	getOwnerIDByOrgID      endpoint.Endpoint
	retrieveRole           endpoint.Endpoint
	viewOrgMembership      endpoint.Endpoint
	viewOrg                endpoint.Endpoint
	assignRole             endpoint.Endpoint
	createDormantOrgInvite endpoint.Endpoint
	activateOrgInvite      endpoint.Endpoint
	timeout                time.Duration
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
		getOwnerIDByOrgID: kitot.TraceClient(tracer, "get_owner_id_by_org_id")(kitgrpc.NewClient(
			conn,
			svcName,
			"GetOwnerIDByOrgID",
			encodeOrgIDRequest,
			decodeOwnerIDResponse,
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
		viewOrgMembership: kitot.TraceClient(tracer, "view_org_membership")(kitgrpc.NewClient(
			conn,
			svcName,
			"ViewOrgMembership",
			encodeViewOrgMembershipRequest,
			decodeOrgMembershipResponse,
			protomfx.OrgMembership{},
		).Endpoint()),
		viewOrg: kitot.TraceClient(tracer, "view_org")(kitgrpc.NewClient(
			conn,
			svcName,
			"ViewOrg",
			encodeViewOrgRequest,
			decodeOrgResponse,
			protomfx.Org{},
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

func (client grpcClient) GetOwnerIDByOrgID(ctx context.Context, req *protomfx.OrgID, opts ...grpc.CallOption) (*protomfx.OwnerID, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.getOwnerIDByOrgID(ctx, orgIDReq{orgID: req.GetValue()})
	if err != nil {
		return nil, err
	}

	oid := res.(ownerIDRes)
	return &protomfx.OwnerID{Value: oid.ownerID}, nil
}

func encodeOrgIDRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(orgIDReq)
	return &protomfx.OrgID{Value: req.orgID}, nil
}

func decodeOwnerIDResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OwnerID)
	return ownerIDRes{ownerID: res.GetValue()}, nil
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

func (client grpcClient) ViewOrgMembership(ctx context.Context, req *protomfx.ViewOrgMembershipReq, _ ...grpc.CallOption) (r *protomfx.OrgMembership, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.viewOrgMembership(ctx, viewOrgMembershipReq{
		token:    req.GetToken(),
		memberID: req.GetMemberID(),
		orgID:    req.GetOrgID(),
	})

	if err != nil {
		return &protomfx.OrgMembership{}, err
	}

	rr := res.(orgMembershipRes)
	return &protomfx.OrgMembership{
		MemberID: rr.memberID,
		OrgID:    rr.orgID,
		Role:     rr.role,
	}, err
}

func encodeViewOrgMembershipRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(viewOrgMembershipReq)
	return &protomfx.ViewOrgMembershipReq{
		Token:    req.token,
		MemberID: req.memberID,
		OrgID:    req.orgID,
	}, nil
}

func decodeOrgMembershipResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.OrgMembership)
	return orgMembershipRes{
		orgID:    res.GetOrgID(),
		memberID: res.GetMemberID(),
		role:     res.GetRole(),
	}, nil
}

func (client grpcClient) ViewOrg(ctx context.Context, req *protomfx.ViewOrgReq, _ ...grpc.CallOption) (r *protomfx.Org, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.viewOrg(ctx, viewOrgReq{
		token: req.GetToken(),
		id:    req.GetId().GetValue(),
	})

	if err != nil {
		return &protomfx.Org{}, err
	}

	rr := res.(orgRes)
	return &protomfx.Org{
		Id:          rr.id,
		OwnerID:     rr.ownerID,
		Name:        rr.name,
		Description: rr.description,
	}, nil
}

func encodeViewOrgRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(viewOrgReq)
	return &protomfx.ViewOrgReq{
		Token: req.token,
		Id: &protomfx.OrgID{
			Value: req.id,
		},
	}, nil
}

func decodeOrgResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.Org)
	return orgRes{
		id:          res.GetId(),
		ownerID:     res.GetOwnerID(),
		name:        res.GetName(),
		description: res.GetDescription(),
	}, nil
}

func (client grpcClient) CreateDormantOrgInvite(ctx context.Context, req *protomfx.CreateDormantOrgInviteReq, _ ...grpc.CallOption) (r *emptypb.Empty, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.createDormantOrgInvite(ctx, createDormantOrgInviteReq{
		token:            req.GetToken(),
		orgID:            req.GetOrgID(),
		inviteeRole:      req.GetInviteeRole(),
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
	return &protomfx.CreateDormantOrgInviteReq{
		Token:            req.token,
		OrgID:            req.orgID,
		InviteeRole:      req.inviteeRole,
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

func decodeEmptyResponse(_ context.Context, _ any) (any, error) {
	return emptyRes{}, nil
}
