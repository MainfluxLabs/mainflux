// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package http

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/metrics"
)

var _ users.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     users.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc users.Service, counter metrics.Counter, latency metrics.Histogram) users.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) SelfRegister(ctx context.Context, user users.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "self_register").Add(1)
		ms.latency.With("method", "self_register").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SelfRegister(ctx, user)
}

func (ms *metricsMiddleware) RegisterAdmin(ctx context.Context, user users.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_admin").Add(1)
		ms.latency.With("method", "register_admin").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RegisterAdmin(ctx, user)
}

func (ms *metricsMiddleware) Register(ctx context.Context, token string, user users.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register").Add(1)
		ms.latency.With("method", "register").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Register(ctx, token, user)
}

func (ms *metricsMiddleware) Login(ctx context.Context, user users.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "login").Add(1)
		ms.latency.With("method", "login").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Login(ctx, user)
}

func (ms *metricsMiddleware) ViewUser(ctx context.Context, token, id string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_user").Add(1)
		ms.latency.With("method", "view_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewUser(ctx, token, id)
}

func (ms *metricsMiddleware) ViewProfile(ctx context.Context, token string) (users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewProfile(ctx, token)
}

func (ms *metricsMiddleware) ListUsers(ctx context.Context, token string, pm users.PageMetadata) (users.UserPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_users").Add(1)
		ms.latency.With("method", "list_users").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListUsers(ctx, token, pm)
}

func (ms *metricsMiddleware) ListUsersByIDs(ctx context.Context, ids []string) (users.UserPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_users_by_ids").Add(1)
		ms.latency.With("method", "list_users_by_ids").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListUsersByIDs(ctx, ids)
}

func (ms *metricsMiddleware) ListUsersByEmails(ctx context.Context, emails []string) ([]users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_users_by_emails").Add(1)
		ms.latency.With("method", "list_users_by_emails").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListUsersByEmails(ctx, emails)
}

func (ms *metricsMiddleware) UpdateUser(ctx context.Context, token string, u users.User) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_user").Add(1)
		ms.latency.With("method", "update_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateUser(ctx, token, u)
}

func (ms *metricsMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "generate_reset_token").Add(1)
		ms.latency.With("method", "generate_reset_token").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GenerateResetToken(ctx, email, host)
}

func (ms *metricsMiddleware) ChangePassword(ctx context.Context, email, password, oldPassword string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "change_password").Add(1)
		ms.latency.With("method", "change_password").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ChangePassword(ctx, email, password, oldPassword)
}

func (ms *metricsMiddleware) ResetPassword(ctx context.Context, email, password string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "reset_password").Add(1)
		ms.latency.With("method", "reset_password").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ResetPassword(ctx, email, password)
}

func (ms *metricsMiddleware) SendPasswordReset(ctx context.Context, host, email, token string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_password_reset").Add(1)
		ms.latency.With("method", "send_password_reset").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SendPasswordReset(ctx, host, email, token)
}

func (ms *metricsMiddleware) EnableUser(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_user").Add(1)
		ms.latency.With("method", "enable_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.EnableUser(ctx, token, id)
}

func (ms *metricsMiddleware) DisableUser(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_user").Add(1)
		ms.latency.With("method", "disable_user").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.DisableUser(ctx, token, id)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) (users.User, []users.User, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) Restore(ctx context.Context, token string, admin users.User, users []users.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore").Add(1)
		ms.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Restore(ctx, token, admin, users)
}
