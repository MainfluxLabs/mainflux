package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/rules"
)

var _ rules.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    rules.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc rules.Service, logger log.Logger) rules.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm loggingMiddleware) CreateRules(ctx context.Context, token string, rules ...rules.Rule) (saved []rules.Rule, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_rules for rules %v took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateRules(ctx, token, rules...)
}

func (lm loggingMiddleware) ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_rules_by_profile for id %s took %s to complete", profileID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRulesByProfile(ctx, token, profileID, pm)
}

func (lm loggingMiddleware) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_rules_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRulesByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ViewRule(ctx context.Context, token, id string) (_ rules.Rule, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_rule for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewRule(ctx, token, id)
}

func (lm loggingMiddleware) UpdateRule(ctx context.Context, token string, rule rules.Rule) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_rule for id %s took %s to complete", rule.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateRule(ctx, token, rule)
}

func (lm loggingMiddleware) RemoveRules(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_rules took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveRules(ctx, token, ids...)
}

func (lm loggingMiddleware) Publish(ctx context.Context, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, msg)
}
