// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func inviteRegistrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(registerByInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		userID, err := svc.RegisterByInvite(ctx, req.User, req.inviteID, req.RedirectPath)
		if err != nil {
			return nil, err
		}

		return createUserRes{created: true, ID: userID}, nil
	}
}

func createPlatformInviteEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createPlatformInviteRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		orgInvite := auth.OrgInvite{
			OrgID:        req.OrgID,
			InviteeRole:  req.Role,
			GroupInvites: req.GroupInvites,
		}

		invite, err := svc.CreatePlatformInvite(ctx, req.token, req.RedirectPath, req.Email, orgInvite)
		if err != nil {
			return nil, err
		}

		return createPlatformInviteRes{
			ID:      invite.ID,
			created: true,
		}, nil
	}
}

func listPlatformInvitesEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listPlatformInvitesRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListPlatformInvites(ctx, req.token, req.pm)
		if err != nil {
			return nil, err
		}

		response := platformInvitePageRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Invites: []platformInviteRes{},
		}

		for _, inv := range page.Invites {
			invRes := buildPlatformInviteResponse(inv)
			response.Invites = append(response.Invites, invRes)
		}

		return response, nil
	}
}

func viewPlatformInviteEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.ViewPlatformInvite(ctx, req.token, req.inviteID)
		if err != nil {
			return nil, err
		}

		return buildPlatformInviteResponse(invite), nil
	}
}

func revokePlatformInviteEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokePlatformInvite(ctx, req.token, req.inviteID); err != nil {
			return nil, err
		}

		return revokePlatformInviteRes{}, nil
	}
}

func buildPlatformInviteResponse(invite users.PlatformInvite) platformInviteRes {
	inviteRes := platformInviteRes{
		ID:           invite.ID,
		InviteeEmail: invite.InviteeEmail,
		CreatedAt:    invite.CreatedAt,
		ExpiresAt:    invite.ExpiresAt,
		State:        invite.State,
	}

	if invite.OrgInvite != nil {
		inviteRes.OrgInvite = &dormantOrgInvite{
			ID:           invite.OrgInvite.ID,
			OrgID:        invite.OrgInvite.OrgID,
			OrgName:      invite.OrgInvite.OrgName,
			InviteeRole:  invite.OrgInvite.InviteeRole,
			GroupInvites: invite.OrgInvite.GroupInvites,
		}
	}

	return inviteRes
}
