package emailer

import (
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	log "github.com/MainfluxLabs/mainflux/logger"
)

type loggingMiddleware struct {
	emailer auth.Emailer
	logger  log.Logger
}

var _ auth.Emailer = (*loggingMiddleware)(nil)

func LoggingMiddleware(e auth.Emailer, logger log.Logger) auth.Emailer {
	return &loggingMiddleware{e, logger}
}

func (lm *loggingMiddleware) SendOrgInvite(to []string, inv auth.OrgInvite, orgName, invRedirectPath string, groupNames map[string]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Emailer method send_org_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.emailer.SendOrgInvite(to, inv, orgName, invRedirectPath, groupNames)
}
