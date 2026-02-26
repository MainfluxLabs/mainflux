package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
)

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

func createRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createRulesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var rulesList []rules.Rule
		for _, rReq := range req.Rules {
			r := rules.Rule{
				Name:        rReq.Name,
				Conditions:  rReq.Conditions,
				Operator:    rReq.Operator,
				Actions:     rReq.Actions,
				Description: rReq.Description,
			}
			rulesList = append(rulesList, r)
		}

		rules, err := svc.CreateRules(ctx, req.token, req.groupID, rulesList...)
		if err != nil {
			return nil, err
		}

		return buildRulesResponse(rules, true), nil
	}
}

func listRulesByThingEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listRulesByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListRulesByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildRulesPageResponse(page, req.pageMetadata), nil
	}
}

func listRulesByGroupEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listRulesByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListRulesByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildRulesPageResponse(page, req.pageMetadata), nil
	}
}

func listThingIDsByRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		ids, err := svc.ListThingIDsByRule(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := thingIDsRes{ThingIDs: ids}
		return res, nil
	}
}

func viewRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		rule, err := svc.ViewRule(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildRuleResponse(rule, false), nil
	}
}

func updateRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateRuleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		rule := rules.Rule{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Conditions:  req.Conditions,
			Operator:    req.Operator,
			Actions:     req.Actions,
		}

		if err := svc.UpdateRule(ctx, req.token, rule); err != nil {
			return nil, err
		}

		return ruleResponse{updated: true}, nil
	}
}

func removeRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeRulesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveRules(ctx, req.token, req.RuleIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func assignRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingRulesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignRules(ctx, req.token, req.thingID, req.RuleIDs...); err != nil {
			return nil, err
		}

		return thingRulesRes{}, nil
	}
}

func unassignRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingRulesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignRules(ctx, req.token, req.thingID, req.RuleIDs...); err != nil {
			return nil, err
		}

		return thingRulesRes{}, nil
	}
}

func createScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var reqScripts []rules.LuaScript
		for _, sReq := range req.Scripts {
			script := rules.LuaScript{
				Name:        sReq.Name,
				Script:      sReq.Script,
				Description: sReq.Description,
			}

			reqScripts = append(reqScripts, script)
		}

		scripts, err := svc.CreateScripts(ctx, req.token, req.groupID, reqScripts...)
		if err != nil {
			return nil, err
		}

		res := buildScriptsResponse(scripts, true)
		return res, nil
	}
}

func listScriptsByThingEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptsPageResponse(page, req.pageMetadata), nil
	}
}

func listScriptsByGroupEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptsPageResponse(page, req.pageMetadata), nil
	}
}

func listThingIDsByScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(scriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		ids, err := svc.ListThingIDsByScript(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := thingIDsRes{ThingIDs: ids}
		return res, nil
	}
}

func viewScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(scriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script, err := svc.ViewScript(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildScriptResponse(script, false), nil
	}
}

func updateScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateScriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script := rules.LuaScript{
			ID:          req.id,
			Name:        req.Name,
			Script:      req.Script,
			Description: req.Description,
		}

		if err := svc.UpdateScript(ctx, req.token, script); err != nil {
			return nil, err
		}

		return scriptRes{updated: true}, nil
	}
}

func removeScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveScripts(ctx, req.token, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func assignScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignScripts(ctx, req.token, req.thingID, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return thingRulesRes{}, nil
	}
}

func unassignScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignScripts(ctx, req.token, req.thingID, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return thingRulesRes{}, nil
	}
}

func buildRulesResponse(rules []rules.Rule, created bool) rulesRes {
	res := rulesRes{Rules: []ruleResponse{}, created: created}

	for _, r := range rules {
		rule := ruleResponse{
			ID:          r.ID,
			GroupID:     r.GroupID,
			Name:        r.Name,
			Description: r.Description,
			Conditions:  r.Conditions,
			Operator:    r.Operator,
			Actions:     r.Actions,
		}
		res.Rules = append(res.Rules, rule)
	}

	return res
}

func buildRulesPageResponse(page rules.RulesPage, pm apiutil.PageMetadata) RulesPageRes {
	res := RulesPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Rules: []ruleResponse{},
	}

	for _, r := range page.Rules {
		rule := ruleResponse{
			ID:          r.ID,
			GroupID:     r.GroupID,
			Name:        r.Name,
			Description: r.Description,
			Conditions:  r.Conditions,
			Operator:    r.Operator,
			Actions:     r.Actions,
		}
		res.Rules = append(res.Rules, rule)
	}

	return res
}

func buildRuleResponse(rule rules.Rule, updated bool) ruleResponse {
	return ruleResponse{
		ID:          rule.ID,
		GroupID:     rule.GroupID,
		Name:        rule.Name,
		Description: rule.Description,
		Conditions:  rule.Conditions,
		Operator:    rule.Operator,
		Actions:     rule.Actions,
		updated:     updated,
	}
}

func buildScriptsResponse(scripts []rules.LuaScript, created bool) scriptsRes {
	res := scriptsRes{Scripts: []scriptRes{}, created: created}

	for _, s := range scripts {
		sr := scriptRes{
			ID:          s.ID,
			GroupID:     s.GroupID,
			Name:        s.Name,
			Script:      s.Script,
			Description: s.Description,
		}
		res.Scripts = append(res.Scripts, sr)
	}

	return res
}

func buildScriptsPageResponse(page rules.LuaScriptsPage, pm apiutil.PageMetadata) scriptsPageRes {
	res := scriptsPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Scripts: []scriptRes{},
	}

	for _, s := range page.Scripts {
		sr := scriptRes{
			ID:          s.ID,
			GroupID:     s.GroupID,
			Name:        s.Name,
			Script:      s.Script,
			Description: s.Description,
		}
		res.Scripts = append(res.Scripts, sr)
	}

	return res
}

func buildScriptResponse(s rules.LuaScript, updated bool) scriptRes {
	return scriptRes{
		ID:          s.ID,
		GroupID:     s.GroupID,
		Name:        s.Name,
		Script:      s.Script,
		Description: s.Description,
		updated:     updated,
	}
}

func listScriptRunsByThingEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptRunsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptRunsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptRunsPageResponse(page, req.pageMetadata), nil
	}
}

func removeScriptRunsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeScriptRunsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveScriptRuns(ctx, req.token, req.ScriptRunIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildScriptRunsPageResponse(page rules.ScriptRunsPage, pm apiutil.PageMetadata) scriptRunsPageRes {
	res := scriptRunsPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Runs: []scriptRunRes{},
	}

	for _, run := range page.Runs {
		sr := scriptRunRes{
			ID:         run.ID,
			ScriptID:   run.ScriptID,
			ThingID:    run.ThingID,
			Logs:       run.Logs,
			StartedAt:  run.StartedAt,
			FinishedAt: run.FinishedAt,
			Status:     run.Status,
			Error:      run.Error,
		}
		res.Runs = append(res.Runs, sr)
	}

	return res
}
