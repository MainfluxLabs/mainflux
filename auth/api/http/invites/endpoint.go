package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
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

		return inviteRes{}, nil
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
