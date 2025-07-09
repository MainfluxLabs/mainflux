package invites

import (
	"context"
	"log"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func inviteMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(invitesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		log.Printf("inviteMembersEndpoint: %+v\n", req)

		if err := svc.InviteMembers(ctx, req.token, req.orgID, req.OrgMembers...); err != nil {
			return nil, err
		}

		return inviteRes{}, nil
	}
}

func revokeInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(revokeInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokeInvite(ctx, req.token, req.inviteID); err != nil {
			return revokeInviteRes{}, err
		}

		return revokeInviteRes{}, nil
	}
}
