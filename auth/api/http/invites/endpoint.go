package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func createOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createOrgInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if _, err := svc.InviteOrgMember(ctx, req.token, req.orgID, req.RedirectPath, req.OrgMember); err != nil {
			return nil, err
		}

		return createOrgInviteRes{}, nil
	}
}

func viewOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewOrgInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.ViewOrgInvite(ctx, req.token, req.inviteID)
		if err != nil {
			return nil, err
		}

		return orgInviteRes{
			ID:          invite.ID,
			InviteeID:   invite.InviteeID,
			OrgID:       invite.OrgID,
			InviterID:   invite.InviterID,
			InviteeRole: invite.InviteeRole,
			CreatedAt:   invite.CreatedAt,
			ExpiresAt:   invite.ExpiresAt,
			State:       invite.State,
		}, nil
	}
}

func revokeOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgInviteRevokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokeOrgInvite(ctx, req.token, req.inviteID); err != nil {
			return nil, err
		}

		return revokeOrgInviteRes{}, nil
	}
}

func respondOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgInviteResponseReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RespondOrgInvite(ctx, req.token, req.inviteID, req.inviteAccepted); err != nil {
			return nil, err
		}

		return respondOrgInviteRes{accept: req.inviteAccepted}, nil
	}
}

func listOrgInvitesByUserEndpoint(svc auth.Service, userType string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgInvitesByUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgInvitesByUser(ctx, req.token, userType, req.userID, req.pm)
		if err != nil {
			return nil, err
		}

		response := orgInvitePageRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Invites: []orgInviteRes{},
		}

		for _, inv := range page.Invites {
			resInv := orgInviteRes{
				ID:          inv.ID,
				InviteeID:   inv.InviteeID,
				InviteeRole: inv.InviteeRole,
				InviterID:   inv.InviterID,
				OrgID:       inv.OrgID,
				CreatedAt:   inv.CreatedAt,
				ExpiresAt:   inv.ExpiresAt,
				State:       inv.State,
			}

			response.Invites = append(response.Invites, resInv)
		}

		return response, nil
	}
}

func listOrgInvitesByOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgInvitesByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgInvitesByOrgID(ctx, req.token, req.orgID, req.pm)
		if err != nil {
			return nil, err
		}

		response := orgInvitePageRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Invites: []orgInviteRes{},
		}

		for _, inv := range page.Invites {
			resInv := orgInviteRes{
				ID:          inv.ID,
				InviteeID:   inv.InviteeID,
				InviteeRole: inv.InviteeRole,
				InviterID:   inv.InviterID,
				OrgID:       inv.OrgID,
				CreatedAt:   inv.CreatedAt,
				ExpiresAt:   inv.ExpiresAt,
				State:       inv.State,
			}

			response.Invites = append(response.Invites, resInv)
		}

		return response, nil
	}
}

func createPlatformInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createPlatformInviteRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.InvitePlatformMember(ctx, req.token, req.RedirectPath, req.Email)
		if err != nil {
			return nil, err
		}

		return platformInviteRes{
			ID:           invite.ID,
			InviteeEmail: invite.InviteeEmail,
			CreatedAt:    invite.CreatedAt,
			ExpiresAt:    invite.ExpiresAt,
			State:        invite.State,
		}, nil
	}
}

func listPlatformInvitesEndpoint(svc auth.Service) endpoint.Endpoint {
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
			resInv := platformInviteRes{
				ID:           inv.ID,
				InviteeEmail: inv.InviteeEmail,
				CreatedAt:    inv.CreatedAt,
				ExpiresAt:    inv.ExpiresAt,
				State:        inv.State,
			}

			response.Invites = append(response.Invites, resInv)
		}

		return response, nil
	}
}

func viewPlatformInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewPlatformInviteRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.ViewPlatformInvite(ctx, req.token, req.inviteID)
		if err != nil {
			return nil, err
		}

		return platformInviteRes{
			ID:           invite.ID,
			InviteeEmail: invite.InviteeEmail,
			CreatedAt:    invite.CreatedAt,
			ExpiresAt:    invite.ExpiresAt,
			State:        invite.State,
		}, nil
	}
}

func revokePlatformInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(revokePlatformInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokePlatformInvite(ctx, req.token, req.inviteID); err != nil {
			return nil, err
		}

		return revokePlatformInviteRes{}, nil
	}
}
