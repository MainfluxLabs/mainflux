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

		invite, err := svc.CreateOrgInvite(ctx, req.token, req.Email, req.Role, req.orgID, req.GroupInvites, req.RedirectPath)
		if err != nil {
			return nil, err
		}

		return createOrgInviteRes{
			created: true,
			ID:      invite.ID,
		}, nil
	}
}

func viewOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		invite, err := svc.ViewOrgInvite(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := buildOrgInviteRes(invite)

		return res, nil
	}
}

func revokeOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(inviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RevokeOrgInvite(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return revokeOrgInviteRes{}, nil
	}
}

func respondOrgInviteEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(respondOrgInviteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RespondOrgInvite(ctx, req.token, req.id, req.accepted); err != nil {
			return nil, err
		}

		return respondOrgInviteRes{accept: req.accepted}, nil
	}
}

func listOrgInvitesByUserEndpoint(svc auth.Service, userType string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgInvitesByUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgInvitesByUser(ctx, req.token, userType, req.id, req.pm)
		if err != nil {
			return nil, err
		}

		response := buildOrgInvitesPageRes(page, req.pm)

		return response, nil
	}
}

func listOrgInvitesByOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgInvitesByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgInvitesByOrg(ctx, req.token, req.id, req.pm)
		if err != nil {
			return nil, err
		}

		response := buildOrgInvitesPageRes(page, req.pm)

		return response, nil
	}
}

func buildOrgInvitesPageRes(page auth.OrgInvitesPage, pm auth.PageMetadataInvites) orgInvitePageRes {
	response := orgInvitePageRes{
		pageRes: pageRes{
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Total:  page.Total,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			State:  pm.State,
		},
		Invites: make([]orgInviteRes, 0, len(page.Invites)),
	}

	for _, inv := range page.Invites {
		response.Invites = append(response.Invites, buildOrgInviteRes(inv))
	}

	return response
}

func buildOrgInviteRes(inv auth.OrgInvite) orgInviteRes {
	return orgInviteRes{
		ID:           inv.ID,
		InviteeID:    inv.InviteeID,
		InviteeEmail: inv.InviteeEmail,
		InviteeRole:  inv.InviteeRole,
		InviterID:    inv.InviterID,
		InviterEmail: inv.InviterEmail,
		OrgID:        inv.OrgID,
		OrgName:      inv.OrgName,
		GroupInvites: inv.GroupInvites,
		CreatedAt:    inv.CreatedAt,
		ExpiresAt:    inv.ExpiresAt,
		State:        inv.State,
	}
}
