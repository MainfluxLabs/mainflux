package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
)

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

func createRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

		rules, err := svc.CreateRules(ctx, req.token, req.profileID, rulesList...)
		if err != nil {
			return nil, err
		}

		return buildRulesResponse(rules, true), nil
	}
}

func listRulesByProfileEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRulesByProfileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListRulesByProfile(ctx, req.token, req.profileID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildRulesPageResponse(page), nil
	}
}

func listRulesByGroupEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listRulesByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListRulesByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildRulesPageResponse(page), nil
	}
}

func viewRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewRuleReq)
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateRuleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		rule := rules.Rule{
			ID:          req.id,
			ProfileID:   req.ProfileID,
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

func buildRulesResponse(rules []rules.Rule, created bool) rulesRes {
	res := rulesRes{Rules: []ruleResponse{}, created: created}

	for _, r := range rules {
		rule := ruleResponse{
			ID:          r.ID,
			ProfileID:   r.ProfileID,
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

func buildRulesPageResponse(page rules.RulesPage) RulesPageRes {
	res := RulesPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: page.Offset,
			Limit:  page.Limit,
		},
		Rules: []ruleResponse{},
	}

	for _, r := range page.Rules {
		rule := ruleResponse{
			ID:          r.ID,
			ProfileID:   r.ProfileID,
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
		ProfileID:   rule.ProfileID,
		GroupID:     rule.GroupID,
		Name:        rule.Name,
		Description: rule.Description,
		Conditions:  rule.Conditions,
		Operator:    rule.Operator,
		Actions:     rule.Actions,
		updated:     updated,
	}
}
