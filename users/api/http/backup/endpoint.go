package backup

import (
	"context"

	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		admin, users, identities, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return buildBackupResponse(admin, users, identities), nil
	}
}

func restoreEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		admin, users, identities := buildBackup(req)

		err := svc.Restore(ctx, req.token, admin, users, identities)
		if err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(admin users.User, users []users.User, identities []users.Identity) backupRes {
	res := backupRes{
		Admin: backupUserRes{
			ID:       admin.ID,
			Email:    admin.Email,
			Password: admin.Password,
			Metadata: admin.Metadata,
			Status:   admin.Status,
		},
		Users: []backupUserRes{},
	}

	for _, user := range users {
		view := backupUserRes{
			ID:       user.ID,
			Email:    user.Email,
			Password: user.Password,
			Metadata: user.Metadata,
			Status:   user.Status,
		}
		res.Users = append(res.Users, view)
	}

	for _, id := range identities {
		res.Identities = append(res.Identities, backupIdentityRes{
			UserID:         id.UserID,
			Provider:       id.Provider,
			ProviderUserID: id.ProviderUserID,
		})
	}

	return res
}

func buildBackup(req restoreReq) (users.User, []users.User, []users.Identity) {
	admin := users.User{
		ID:       req.Admin.ID,
		Email:    req.Admin.Email,
		Password: req.Admin.Password,
		Metadata: req.Admin.Metadata,
		Status:   req.Admin.Status,
	}

	u := []users.User{}
	for _, user := range req.Users {
		view := users.User{
			ID:       user.ID,
			Email:    user.Email,
			Password: user.Password,
			Metadata: user.Metadata,
			Status:   user.Status,
		}
		u = append(u, view)
	}

	var identities []users.Identity
	for _, id := range req.Identities {
		identities = append(identities, users.Identity{
			UserID:         id.UserID,
			Provider:       id.Provider,
			ProviderUserID: id.ProviderUserID,
		})
	}

	return admin, u, identities
}
