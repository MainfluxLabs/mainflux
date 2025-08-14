package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func createInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if _, err := svc.InviteMember(ctx, req.token, req.orgID, req.RedirectPathInvite, req.RedirectPathRegister, req.OrgMember); err != nil {
			return nil, err
		}

		return createInviteRes{}, nil
	}
}

func viewInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.ViewInvite(ctx, req.token, req.inviteID)
		if err != nil {
			return nil, err
		}

		return inviteRes{
			ID:           invite.ID,
			InviteeID:    invite.InviteeID,
			InviteeEmail: invite.InviteeEmail,
			OrgID:        invite.OrgID,
			InviterID:    invite.InviterID,
			InviteeRole:  invite.InviteeRole,
			CreatedAt:    invite.CreatedAt,
			ExpiresAt:    invite.ExpiresAt,
		}, nil
	}
}

func revokeInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteRevokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokeInvite(ctx, req.token, req.inviteID); err != nil {
			return nil, err
		}

		return revokeInviteRes{}, nil
	}
}

func respondInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteResponseReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.InviteRespond(ctx, req.token, req.inviteID, req.inviteAccepted); err != nil {
			return nil, err
		}

		return respondInviteRes{accept: req.inviteAccepted}, nil
	}
}

func listInvitesByUserEndpoint(svc auth.Service, userType string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listInvitesByUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListInvitesByUser(ctx, req.token, userType, req.userID, req.pm)
		if err != nil {
			return nil, err
		}

		response := invitePageRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Invites: []inviteRes{},
		}

		for _, inv := range page.Invites {
			resInv := inviteRes{
				ID:          inv.ID,
				InviteeID:   inv.InviteeID,
				InviteeRole: inv.InviteeRole,
				InviterID:   inv.InviterID,
				OrgID:       inv.OrgID,
				CreatedAt:   inv.CreatedAt,
				ExpiresAt:   inv.ExpiresAt,
			}

			response.Invites = append(response.Invites, resInv)
		}

		return response, nil
	}
}
