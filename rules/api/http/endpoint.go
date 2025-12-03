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
