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
