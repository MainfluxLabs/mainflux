package emailer

import (
	"time"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/metrics"
)

var _ things.Emailer = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	emailer things.Emailer
}

func MetricsMiddleware(emailer things.Emailer, counter metrics.Counter, latency metrics.Histogram) things.Emailer {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		emailer: emailer,
	}
}

func (ms *metricsMiddleware) SendGroupMembershipNotification(to []string, orgName, groupName, groupRole string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "send_group_membership_notification").Add(1)
		ms.latency.With("method", "send_group_membership_notification").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.emailer.SendGroupMembershipNotification(to, orgName, groupName, groupRole)
}
