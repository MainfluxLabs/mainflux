package emailer

import (
	"time"

	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/metrics"
)

var _ users.Emailer = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	emailer users.Emailer
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(emailer users.Emailer, counter metrics.Counter, latency metrics.Histogram) users.Emailer {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		emailer: emailer,
	}
}

func (ms *metricsMiddleware) SendPasswordReset(To []string, redirectPath, token string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_password_reset").Add(1)
		ms.latency.With("method", "send_password_reset").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.emailer.SendPasswordReset(To, redirectPath, token)
}

func (ms *metricsMiddleware) SendEmailVerification(To []string, redirectPath, token string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_email_verification").Add(1)
		ms.latency.With("method", "send_email_verification").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.emailer.SendEmailVerification(To, redirectPath, token)
}

func (ms *metricsMiddleware) SendPlatformInvite(To []string, invite users.PlatformInvite, redirectPath string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_platform_invite").Add(1)
		ms.latency.With("method", "send_platform_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.emailer.SendPlatformInvite(To, invite, redirectPath)
}
