package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/endpoint"
)

func inviteMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(invitesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.InviteMembers(ctx, req.token, req.orgID, req.OrgMembers...); err != nil {
			return nil, err
		}

		return createInviteRes{}, nil
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

		// TODO: perhaps this endpoint should return the ID of the org the user has just been assigned
		// to or something?
		return respondInviteRes{}, nil
	}
}

func listInvitesByUserEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listInvitesByUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := apiutil.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}

		page, err := svc.ListInvitesByInviteeID(ctx, req.token, req.userID, pm)
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
