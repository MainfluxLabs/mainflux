package emailer

import (
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/things"
)

type loggingMiddleware struct {
	emailer things.Emailer
	logger  log.Logger
}

var _ things.Emailer = (*loggingMiddleware)(nil)

func LoggingMiddleware(e things.Emailer, logger log.Logger) things.Emailer {
	return &loggingMiddleware{e, logger}
}

func (lm *loggingMiddleware) SendGroupInvite(to []string, inv things.GroupInvite, orgName, invRedirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Emailer method send_group_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.emailer.SendGroupInvite(to, inv, orgName, invRedirectPath)
}
