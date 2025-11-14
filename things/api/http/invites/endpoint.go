package invites

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createGroupInviteEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createGroupInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		inv, err := svc.CreateGroupInvite(ctx, req.token, req.Email, req.Role, req.groupID, req.RedirectPath)
		if err != nil {
			return nil, err
		}

		return createGroupInviteRes{
			created: true,
			ID:      inv.ID,
		}, nil
	}
}

func viewGroupInviteEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		inv, err := svc.ViewGroupInvite(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildGroupInviteRes(inv), nil
	}
}

func revokeGroupInviteEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokeGroupInvite(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return revokeGroupInviteRes{}, nil
	}
}

func respondGroupInviteEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(respondGroupInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RespondGroupInvite(ctx, req.token, req.id, req.accepted); err != nil {
			return nil, err
		}

		return respondGroupInviteRes{accept: req.accepted}, nil
	}
}

func listGroupInvitesByUserEndpoint(svc things.Service, userType string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupInvitesByUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroupInvitesByUser(ctx, req.token, userType, req.id, req.pm)
		if err != nil {
			return nil, err
		}

		return buildGroupInvitesPageRes(page, req.pm), nil
	}
}

func listGroupInvitesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupInvitesByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroupInvitesByGroup(ctx, req.token, req.id, req.pm)
		if err != nil {
			return nil, err
		}

		return buildGroupInvitesPageRes(page, req.pm), nil
	}
}

func listGroupInvitesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupInvitesByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// TODO: create svc method
		page, err := svc.ListGroupInvitesByGroup(ctx, req.token, req.id, req.pm)
		if err != nil {
			return nil, err
		}

		return buildGroupInvitesPageRes(page, req.pm), nil
	}
}
