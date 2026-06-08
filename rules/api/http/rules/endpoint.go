// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"

	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
)

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
				Description: rReq.Description,
				Input:       rReq.Input,
				Conditions:  rReq.Conditions,
				Operator:    rReq.Operator,
				Actions:     rReq.Actions,
			}
			rulesList = append(rulesList, r)
		}

		rules, err := svc.CreateRules(ctx, req.token, req.groupID, rulesList...)
		if err != nil {
			return nil, err
		}

		res := rulesRes{Rules: []ruleResponse{}, created: true}
		for _, r := range rules {
			res.Rules = append(res.Rules, toRuleResponse(r))
		}
		return res, nil
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

		return toRuleResponse(rule), nil
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
			Input:       rules.Input{Type: req.Input.Type},
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

func assignThingsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ruleThingsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignThings(ctx, req.token, req.ruleID, req.ThingIDs...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignThingsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(ruleThingsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignThings(ctx, req.token, req.ruleID, req.ThingIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
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

func toRuleResponse(r rules.Rule) ruleResponse {
	return ruleResponse{
		ID:          r.ID,
		GroupID:     r.GroupID,
		Name:        r.Name,
		Description: r.Description,
		Input:       r.Input,
		Conditions:  r.Conditions,
		Operator:    r.Operator,
		Actions:     r.Actions,
	}
}

func buildRulesPageResponse(page rules.RulesPage, pm rules.PageMetadata) RulesPageRes {
	res := RulesPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Rules: []ruleResponse{},
	}

	for _, r := range page.Rules {
		res.Rules = append(res.Rules, toRuleResponse(r))
	}

	return res
}
