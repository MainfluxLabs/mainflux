// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:     req.keyType,
			Subject:  req.email,
			IssuerID: req.id,
			IssuedAt: time.Now().UTC(),
		}

		_, secret, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}

		return issueRes{secret}, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		ret := identityRes{
			id:    id.ID,
			email: id.Email,
		}

		return ret, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		ar := auth.AuthzReq{
			Token:   req.Token,
			Object:  req.Object,
			Subject: req.Subject,
			Action:  req.Action,
		}

		if err := svc.Authorize(ctx, ar); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func getOwnerIDByOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ownerIDByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		ownerID, err := svc.GetOwnerIDByOrg(ctx, req.orgID)
		if err != nil {
			return ownerIDByOrgReq{}, err
		}

		return ownerIDByOrgRes{ownerID: ownerID}, nil
	}
}

func assignRoleEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(assignRoleReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		if err := svc.AssignRole(ctx, req.ID, req.Role); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}
func retrieveRoleEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(retrieveRoleReq)

		if err := req.validate(); err != nil {
			return retrieveRoleRes{}, err
		}

		role, err := svc.RetrieveRole(ctx, req.id)
		if err != nil {
			return retrieveRoleRes{}, err
		}

		res := retrieveRoleRes{
			role: role,
		}

		return res, nil
	}
}

func createDormantOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createDormantOrgInviteReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		orgInvite := auth.OrgInvite{
			OrgID:        req.orgID,
			InviteeRole:  req.inviteeRole,
			GroupInvites: req.groupInvites,
		}

		if _, err := svc.CreateDormantOrgInvite(ctx, req.token, orgInvite, req.platformInviteID); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func activateOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(activateOrgInviteReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		err := svc.ActivateOrgInvite(ctx, req.platformInviteID, req.userID, req.redirectPath)
		if err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func getDormantInviteByPlatformInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(getDormantInviteByPlatformInviteReq)
		if err := req.validate(); err != nil {
			return orgInviteRes{}, err
		}

		invite, err := svc.GetDormantInviteByPlatformInvite(ctx, req.platformInviteID)
		if err != nil {
			return orgInviteRes{}, err
		}

		return orgInviteRes{invite}, nil
	}
}

func viewOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		org, err := svc.ViewOrg(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return orgRes{
			id:      org.ID,
			ownerID: org.OwnerID,
			name:    org.Name,
		}, nil
	}
}
