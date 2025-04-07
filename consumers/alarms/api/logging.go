package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ alarms.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    alarms.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc alarms.Service, logger log.Logger) alarms.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm loggingMiddleware) ListAlarmsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_alarms_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAlarmsByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ListAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (_ alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_alarms_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAlarmsByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) ViewAlarm(ctx context.Context, token, id string) (_ alarms.Alarm, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_alarm for id %s took %s to complete", id, time.Since(begin))
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
		message := fmt.Sprintf("Method remove_alarms for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveAlarms(ctx, token, id...)
}

func (lm loggingMiddleware) Consume(alarm interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Consume(alarm)
}
