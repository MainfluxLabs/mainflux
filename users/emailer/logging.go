package emailer

import (
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/users"
)

type loggingMiddleware struct {
	emailer users.Emailer
	logger  log.Logger
}

var _ users.Emailer = (*loggingMiddleware)(nil)

func LoggingMiddleware(e users.Emailer, logger log.Logger) users.Emailer {
	return &loggingMiddleware{e, logger}
}

func (lm *loggingMiddleware) SendPasswordReset(To []string, redirectPath, token string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Emailer method send_password_reset took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.emailer.SendPasswordReset(To, redirectPath, token)
}

func (lm *loggingMiddleware) SendEmailVerification(To []string, redirectPath, token string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Emailer method send_email_verification took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.emailer.SendEmailVerification(To, redirectPath, token)
}

func (lm *loggingMiddleware) SendPlatformInvite(to []string, invite users.PlatformInvite, redirectPath string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Emailer method send_platform_invite took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))

	}(time.Now())

	return lm.emailer.SendPlatformInvite(to, invite, redirectPath)
}
