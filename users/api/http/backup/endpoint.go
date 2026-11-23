package backup

import (
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		platformOwner, users, identities, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return buildBackupResponse(platformOwner, users, identities), nil
	}
}

func restoreEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		platformOwner, users, identities := buildBackup(req)

		if err := svc.Restore(ctx, req.token, platformOwner, users, identities); err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusCreated}, nil
	}
}

func buildBackupResponse(platformOwner users.User, users []users.User, identities []users.Identity) backupRes {
	res := backupRes{
		PlatformOwner: backupUserRes{
			ID:       platformOwner.ID,
			Email:    platformOwner.Email,
			Password: platformOwner.Password,
			Metadata: platformOwner.Metadata,
			Status:   platformOwner.Status,
		},
		Users:      []backupUserRes{},
		Identities: []backupIdentityRes{},
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

	for _, identity := range identities {
		view := backupIdentityRes{
			UserID:         identity.UserID,
			Provider:       identity.Provider,
			ProviderUserID: identity.ProviderUserID,
		}
		res.Identities = append(res.Identities, view)
	}

	return res
}

func buildBackup(req restoreReq) (users.User, []users.User, []users.Identity) {
	platformOwner := users.User{
		ID:       req.PlatformOwner.ID,
		Email:    req.PlatformOwner.Email,
		Password: req.PlatformOwner.Password,
		Metadata: req.PlatformOwner.Metadata,
		Status:   req.PlatformOwner.Status,
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

	i := []users.Identity{}
	for _, identity := range req.Identities {
		view := users.Identity{
			UserID:         identity.UserID,
			Provider:       identity.Provider,
			ProviderUserID: identity.ProviderUserID,
		}
		i = append(i, view)
	}

	return platformOwner, u, i
}
