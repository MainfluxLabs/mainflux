// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package http

import (
	"context"
	"fmt"
	"time"

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

func (lm *loggingMiddleware) SelfRegister(ctx context.Context, user users.User) (uid string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method self_register for user %s took %s to complete", user.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.svc.SelfRegister(ctx, user)
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

func (lm *loggingMiddleware) Register(ctx context.Context, token string, user users.User) (uid string, err error) {
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

func (lm *loggingMiddleware) ListUsers(ctx context.Context, token string, pm users.PageMetadata) (e users.UserPage, err error) {
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

func (lm *loggingMiddleware) ListUsersByIDs(ctx context.Context, ids []string) (u users.UserPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_users_by_ids for ids %s took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListUsersByIDs(ctx, ids)
}

func (lm *loggingMiddleware) ListUsersByEmails(ctx context.Context, emails []string) (u []users.User, err error) {
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

func (lm *loggingMiddleware) GenerateResetToken(ctx context.Context, email, host string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method generate_reset_token for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.GenerateResetToken(ctx, email, host)
}

func (lm *loggingMiddleware) ChangePassword(ctx context.Context, email, password, oldPassword string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method change_password for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ChangePassword(ctx, email, password, oldPassword)
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

func (lm *loggingMiddleware) SendPasswordReset(ctx context.Context, host, email, token string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method send_password_reset for user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SendPasswordReset(ctx, host, email, token)
}

func (lm *loggingMiddleware) EnableUser(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method enable_user for user %s took %s to complete", id, time.Since(begin))
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
		message := fmt.Sprintf("Method disable_user for user %s took %s to complete", id, time.Since(begin))
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
