package emailer

import (
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/metrics"
)

var _ auth.Emailer = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	emailer auth.Emailer
}

func MetricsMiddleware(emailer auth.Emailer, counter metrics.Counter, latency metrics.Histogram) auth.Emailer {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		emailer: emailer,
	}
}

func (ms *metricsMiddleware) SendOrgInvite(to []string, invite auth.OrgInvite, orgName, invRedirectPath string, groupNames map[string]string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_org_invite").Add(1)
		ms.latency.With("method", "send_org_invite").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.emailer.SendOrgInvite(to, invite, orgName, invRedirectPath, groupNames)
}
