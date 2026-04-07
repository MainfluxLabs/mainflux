package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ alarms.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    alarms.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc alarms.Service, logger log.Logger, auth domain.AuthClient) alarms.Service {
	return &loggingMiddleware{logger, svc, auth}
}

func (lm loggingMiddleware) identify(token string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := lm.auth.Identify(ctx, token)
	if err != nil {
		return ""
	}
	return id.Email
}

func (lm loggingMiddleware) ListAlarmsByGroup(ctx context.Context, token, groupID string, pm alarms.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_alarms_by_group by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAlarmsByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ListAlarmsByThing(ctx context.Context, token, thingID string, pm alarms.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_alarms_by_thing by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAlarmsByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) ListAlarmsByOrg(ctx context.Context, token, orgID string, pm alarms.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_alarms_by_org by user %s, org id %s took %s to complete", email, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAlarmsByOrg(ctx, token, orgID, pm)
}

func (lm loggingMiddleware) ViewAlarm(ctx context.Context, token, id string) (_ alarms.Alarm, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method view_alarm by user %s, alarm id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewAlarm(ctx, token, id)
}

func (lm loggingMiddleware) RemoveAlarms(ctx context.Context, token string, id ...string) (err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method remove_alarms by user %s, alarm id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveAlarms(ctx, token, id...)
}

func (lm loggingMiddleware) RemoveAlarmsByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_alarms_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveAlarmsByThing(ctx, thingID)
}

func (lm loggingMiddleware) RemoveAlarmsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_alarms_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveAlarmsByGroup(ctx, groupID)
}

func (lm loggingMiddleware) ExportAlarmsByThing(ctx context.Context, token, thingID string, pm alarms.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method export_alarms_by_thing by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ExportAlarmsByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) Consume(subject string, alarm any) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Consume(subject, alarm)
}
