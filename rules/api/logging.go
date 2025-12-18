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

func (lm loggingMiddleware) CreateRules(ctx context.Context, token, groupID string, rules ...rules.Rule) (saved []rules.Rule, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_rules for rules %v took %s to complete", saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateRules(ctx, token, groupID, rules...)
}

func (lm loggingMiddleware) ListRulesByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_rules_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRulesByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_rules_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRulesByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ListThingIDsByRule(ctx context.Context, token, ruleID string) (_ []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_thing_ids_by_rule for rule id %s took %s to complete", ruleID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingIDsByRule(ctx, token, ruleID)
}

func (lm loggingMiddleware) ViewRule(ctx context.Context, token, id string) (_ rules.Rule, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_rule rule for id %s took %s to complete", id, time.Since(begin))
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
		message := fmt.Sprintf("Method update_rule for rule id %s took %s to complete", rule.ID, time.Since(begin))
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
		message := fmt.Sprintf("Method remove_rules for rule ids %v took %s to complete", ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveRules(ctx, token, ids...)
}

func (lm loggingMiddleware) RemoveRulesByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_rules_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveRulesByGroup(ctx, groupID)
}

func (lm loggingMiddleware) AssignRules(ctx context.Context, token, thingID string, ruleIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method assign_rules took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignRules(ctx, token, thingID, ruleIDs...)
}

func (lm loggingMiddleware) UnassignRules(ctx context.Context, token, thingID string, ruleIDs ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_rules for rule ids %v took %s to complete", ruleIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignRules(ctx, token, thingID, ruleIDs...)
}

func (lm loggingMiddleware) UnassignRulesByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_rules_by_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignRulesByThing(ctx, thingID)
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
