package api

import (
	"context"
	"fmt"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"

	log "github.com/MainfluxLabs/mainflux/logger"
	pkgauth "github.com/MainfluxLabs/mainflux/pkg/auth"
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
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method create_rules by user %s, rules %v took %s to complete", email, saved, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateRules(ctx, token, groupID, rules...)
}

func (lm loggingMiddleware) ListRulesByThing(ctx context.Context, token, thingID string, pm rules.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_rules_by_thing by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRulesByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) ListRulesByGroup(ctx context.Context, token, groupID string, pm rules.PageMetadata) (_ rules.RulesPage, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_rules_by_group by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
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
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_thing_ids_by_rule by user %s, rule id %s took %s to complete", email, ruleID, time.Since(begin))
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
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method view_rule by user %s, rule id %s took %s to complete", email, id, time.Since(begin))
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
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method update_rule by user %s, rule id %s took %s to complete", email, rule.ID, time.Since(begin))
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
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_rules by user %s, rule ids %v took %s to complete", email, ids, time.Since(begin))
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

func (lm loggingMiddleware) ConsumeMessage(subject string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume_message took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConsumeMessage(subject, msg)
}

func (lm loggingMiddleware) CreateScripts(ctx context.Context, token, groupID string, scripts ...rules.LuaScript) (_ []rules.LuaScript, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method create_scripts by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateScripts(ctx, token, groupID, scripts...)
}

func (lm loggingMiddleware) ListScriptsByThing(ctx context.Context, token, thingID string, pm rules.PageMetadata) (_ rules.LuaScriptsPage, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_scripts_by_thing by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListScriptsByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) ListScriptsByGroup(ctx context.Context, token, groupID string, pm rules.PageMetadata) (_ rules.LuaScriptsPage, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_scripts_by_group by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListScriptsByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ListThingIDsByScript(ctx context.Context, token, scriptID string) (_ []string, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_thing_ids_by_script by user %s, script id %s took %s to complete", email, scriptID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThingIDsByScript(ctx, token, scriptID)
}

func (lm loggingMiddleware) ViewScript(ctx context.Context, token, id string) (_ rules.LuaScript, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method view_script by user %s, script id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewScript(ctx, token, id)
}

func (lm loggingMiddleware) UpdateScript(ctx context.Context, token string, script rules.LuaScript) (err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method update_script by user %s, script id %s took %s to complete", email, script.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateScript(ctx, token, script)
}

func (lm loggingMiddleware) RemoveScripts(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_scripts by user %s, script ids %v took %s to complete", email, ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveScripts(ctx, token, ids...)
}

func (lm loggingMiddleware) RemoveScriptsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_scripts_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveScriptsByGroup(ctx, groupID)
}

func (lm loggingMiddleware) AssignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) (err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method assign_scripts by user %s, thing id %s and script ids %v took %s to complete", email, thingID, scriptIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AssignScripts(ctx, token, thingID, scriptIDs...)
}

func (lm loggingMiddleware) UnassignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) (err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method unassign_scripts by user %s, thing id %s and script ids %v took %s to complete", email, thingID, scriptIDs, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignScripts(ctx, token, thingID, scriptIDs...)
}

func (lm loggingMiddleware) UnassignScriptsFromThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unassign_scripts_from_thing for thing id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UnassignScriptsFromThing(ctx, thingID)
}

func (lm loggingMiddleware) ListScriptRunsByThing(ctx context.Context, token, thingID string, pm rules.PageMetadata) (_ rules.ScriptRunsPage, err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method list_script_runs_by_thing by user %s, thing id %s took %s to complete", email, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListScriptRunsByThing(ctx, token, thingID, pm)
}

func (lm loggingMiddleware) RemoveScriptRuns(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		email := pkgauth.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_script_runs by user %s, run ids %v took %s to complete", email, ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}

		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveScriptRuns(ctx, token, ids...)
}
