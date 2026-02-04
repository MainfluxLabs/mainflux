// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"

	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func selfRegistrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(selfRegisterUserReq)
		if err := req.validate(); err != nil {
			return selfRegisterRes{}, err
		}

		_, err := svc.SelfRegister(ctx, req.User, req.RedirectPath)
		if err != nil {
			return selfRegisterRes{}, err
		}

		return selfRegisterRes{}, nil
	}
}

func verifyEmailEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(verifyEmailReq)
		if err := req.validate(); err != nil {
			return createUserRes{}, err
		}

		userID, err := svc.VerifyEmail(ctx, req.emailToken)
		if err != nil {
			return createUserRes{}, err
		}

		return createUserRes{created: true, ID: userID}, nil
	}
}

func registrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(registerUserReq)
		if err := req.validate(); err != nil {
			return createUserRes{}, err
		}
		uid, err := svc.Register(ctx, req.token, req.user)
		if err != nil {
			return createUserRes{}, err
		}
		ucr := createUserRes{
			ID:      uid,
			created: true,
		}

		return ucr, nil
	}
}

// Password reset request endpoint. A client makes a call to this endpoint,
// supplying an e-mail address and a "path" string, which is appended to the
// host of the running service to generate a final password reset link
// that the users receives in an e-mail message:
// URL: {MF_HOST}+{redirect_path}+"?token="+<token>
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwResetReqRes{}
		if err := svc.GenerateResetToken(ctx, req.Email, req.RedirectPath); err != nil {
			return nil, err
		}
		res.Msg = MailSent

		return res, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}
		if err := svc.ResetPassword(ctx, req.Token, req.Password); err != nil {
			return nil, err
		}
		return res, nil
	}
}

func viewUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.ViewUser(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return viewUserRes{
			ID:       u.ID,
			Email:    u.Email,
			Metadata: u.Metadata,
		}, nil
	}
}

func viewProfileEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.ViewProfile(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return viewUserRes{
			ID:       u.ID,
			Email:    u.Email,
			Metadata: u.Metadata,
			Role:     u.Role,
		}, nil
	}
}

func listUsersEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listUsersReq)
		if err := req.validate(); err != nil {
			return users.UserPage{}, err
		}
		pm := users.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Email:    req.email,
			Status:   req.status,
			Metadata: req.metadata,
			Order:    req.order,
			Dir:      req.dir,
		}
		up, err := svc.ListUsers(ctx, req.token, pm)
		if err != nil {
			return users.UserPage{}, err
		}

		return buildUsersResponse(up, pm), nil
	}
}

func updateUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateUserReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		user := users.User{
			Metadata: req.Metadata,
		}
		err := svc.UpdateUser(ctx, req.token, user)
		if err != nil {
			return nil, err
		}
		return updateUserRes{}, nil
	}
}

func passwordChangeEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(passwChangeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res := passwChangeRes{}
		if err := svc.ChangePassword(ctx, req.token, req.Email, req.Password, req.OldPassword); err != nil {
			return nil, err
		}
		return res, nil
	}
}

func loginEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(userReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		token, err := svc.Login(ctx, req.user)
		if err != nil {
			return nil, err
		}

		return tokenRes{token}, nil
	}
}

func oauthLoginEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(oauthLoginReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		state, verifier, redirectURL := svc.OAuthLogin(req.provider)

		return oauthLoginRes{
			State:       state,
			Verifier:    verifier,
			RedirectURL: redirectURL,
		}, nil
	}
}

func oauthCallbackEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(oauthCallbackReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		redirectURL, err := svc.OAuthCallback(ctx, req.provider, req.code, req.verifier)
		if err != nil {
			return nil, err
		}

		return redirectURLRes{redirectURL}, nil
	}
}

func enableUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.EnableUser(ctx, req.token, req.id); err != nil {
			return nil, err
		}
		return deleteRes{}, nil
	}
}

func disableUserEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeUserStatusReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := svc.DisableUser(ctx, req.token, req.id); err != nil {
			return nil, err
		}
		return deleteRes{}, nil
	}
}

func buildUsersResponse(up users.UserPage, pm users.PageMetadata) userPageRes {
	res := userPageRes{
		pageRes: pageRes{
			Total:  up.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Email:  pm.Email,
			Status: pm.Status,
		},
		Users: []viewUserRes{},
	}
	for _, user := range up.Users {
		view := viewUserRes{
			ID:       user.ID,
			Email:    user.Email,
			Metadata: user.Metadata,
		}
		res.Users = append(res.Users, view)
	}
	return res
}
