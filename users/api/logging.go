// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    users.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc users.Service, logger log.Logger) users.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) SelfRegister(ctx context.Context, user users.User, redirectPath string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method self_register for user %s took %s to complete", user.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.SelfRegister(ctx, user, redirectPath)
}

func (lm *loggingMiddleware) VerifyEmail(ctx context.Context, confirmationToken string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method verify_email took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.VerifyEmail(ctx, confirmationToken)
}

func (lm *loggingMiddleware) RegisterByInvite(ctx context.Context, user users.User, inviteID, orgInviteRedirectPath string) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method register_by_invite for user %s and invite id %s took %s to complete", user.Email, inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.RegisterByInvite(ctx, user, inviteID, orgInviteRedirectPath)
}

func (lm *loggingMiddleware) RegisterAdmin(ctx context.Context, user users.User) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method register_admin for user %s took %s to complete", user.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.RegisterAdmin(ctx, user)
}

func (lm *loggingMiddleware) Register(ctx context.Context, token string, user users.User) (_ string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method register for user %s took %s to complete", user.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.Register(ctx, token, user)
}

func (lm *loggingMiddleware) Login(ctx context.Context, user users.User) (token string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method login for user %s and token %s took %s to complete", user.Email, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Login(ctx, user)
}

func (lm *loggingMiddleware) OAuthLogin(provider string) (state, verifier, redirectURL string) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method oauth_login for provider %s took %s to complete", provider, time.Since(begin))
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.OAuthLogin(provider)
}

func (lm *loggingMiddleware) OAuthCallback(ctx context.Context, provider, code, verifier string) (token string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method oauth_callback for provider %s took %s to complete", provider, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.OAuthCallback(ctx, provider, code, verifier)
}

func (lm *loggingMiddleware) ViewUser(ctx context.Context, token, id string) (u users.User, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_user for user %s took %s to complete", u.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewUser(ctx, token, id)
}

func (lm *loggingMiddleware) ViewProfile(ctx context.Context, token string) (u users.User, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_profile for user %s took %s to complete", u.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewProfile(ctx, token)
}

func (lm *loggingMiddleware) ListUsers(ctx context.Context, token string, pm users.PageMetadata) (_ users.UserPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_users took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListUsers(ctx, token, pm)
}

func (lm *loggingMiddleware) ListUsersByIDs(ctx context.Context, ids []string, pm users.PageMetadata) (_ users.UserPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_users_by_ids for ids %s and email %s took %s to complete", ids, pm.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListUsersByIDs(ctx, ids, pm)
}

func (lm *loggingMiddleware) ListUsersByEmails(ctx context.Context, emails []string) (_ []users.User, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_users_by_emails for emails %s took %s to complete", emails, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListUsersByEmails(ctx, emails)
}

func (lm *loggingMiddleware) UpdateUser(ctx context.Context, token string, u users.User) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_user for user %s took %s to complete", u.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateUser(ctx, token, u)
}

func (lm *loggingMiddleware) GenerateResetToken(ctx context.Context, email, redirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method generate_reset_token for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GenerateResetToken(ctx, email, redirectPath)
}

func (lm *loggingMiddleware) ChangePassword(ctx context.Context, token, email, password, oldPassword string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method change_password for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ChangePassword(ctx, token, email, password, oldPassword)
}

func (lm *loggingMiddleware) ResetPassword(ctx context.Context, email, password string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method reset_password for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ResetPassword(ctx, email, password)
}

func (lm *loggingMiddleware) SendPasswordReset(ctx context.Context, redirectPath, email, token string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_password_reset for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendPasswordReset(ctx, redirectPath, email, token)
}

func (lm *loggingMiddleware) EnableUser(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method enable_user for user id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.EnableUser(ctx, token, id)
}

func (lm *loggingMiddleware) DisableUser(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disable_user for user id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisableUser(ctx, token, id)
}

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (users.User, []users.User, error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup took %s to complete", time.Since(begin))
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Backup(ctx, token)
}

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, admin users.User, users []users.User) error {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore took %s to complete", time.Since(begin))
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, admin, users)
}

func (lm *loggingMiddleware) CreatePlatformInvite(ctx context.Context, token, redirectPath, email string, orgInvite auth.OrgInvite) (_ users.PlatformInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_platform_invite for email %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreatePlatformInvite(ctx, token, redirectPath, email, orgInvite)
}

func (lm *loggingMiddleware) RevokePlatformInvite(ctx context.Context, token, inviteID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_platform_invite for invite id %s took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokePlatformInvite(ctx, token, inviteID)
}

func (lm *loggingMiddleware) ViewPlatformInvite(ctx context.Context, inviteID string) (_ users.PlatformInvite, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_platform_invite for invite id %s took %s to complete", inviteID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewPlatformInvite(ctx, inviteID)
}

func (lm *loggingMiddleware) ListPlatformInvites(ctx context.Context, token string, pm users.PageMetadataInvites) (_ users.PlatformInvitesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_platform_invites took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListPlatformInvites(ctx, token, pm)
}

func (lm *loggingMiddleware) ValidatePlatformInvite(ctx context.Context, inviteID string, email string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method validate_platform_invite for invite id %s and email %s took %s to complete", inviteID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ValidatePlatformInvite(ctx, inviteID, email)
}

func (lm *loggingMiddleware) SendPlatformInviteEmail(ctx context.Context, invite users.PlatformInvite, redirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_platform_invite_email for invite id %s took %s to complete", invite.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendPlatformInviteEmail(ctx, invite, redirectPath)
}
