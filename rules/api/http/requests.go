// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

const (
	minLen       = 1
	maxLimitSize = 200
	maxNameSize  = 254
)

type rule struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Conditions  []rules.Condition `json:"conditions"`
	Operator    string            `json:"operator,omitempty"`
	Actions     []rules.Action    `json:"actions"`
}

type createRulesReq struct {
	token   string
	groupID string
	Rules   []rule `json:"rules"`
}

func (req createRulesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Rules) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, rule := range req.Rules {
		if err := rule.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req rule) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if len(req.Conditions) < minLen {
		return apiutil.ErrEmptyList
	}
	for _, condition := range req.Conditions {
		if condition.Field == "" {
			return apiutil.ErrMissingConditionField
		}
		if condition.Comparator == "" {
			return apiutil.ErrMissingConditionComparator
		}
		if condition.Threshold == nil {
			return apiutil.ErrMissingConditionThreshold
		}
	}

	if len(req.Conditions) > minLen {
		if req.Operator != rules.OperatorAND && req.Operator != rules.OperatorOR {
			return apiutil.ErrInvalidOperator
		}
	}

	if len(req.Actions) < minLen {
		return apiutil.ErrEmptyList
	}
	for _, action := range req.Actions {
		switch action.Type {
		case rules.ActionTypeSMTP, rules.ActionTypeSMPP:
			if action.ID == "" {
				return apiutil.ErrMissingActionID
			}
		case rules.ActionTypeAlarm:
		default:
			return apiutil.ErrInvalidActionType
		}
	}

	return nil
}

type ruleReq struct {
	token string
	id    string
}

func (req ruleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingRuleID
	}

	return nil
}

type listRulesByThingReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listRulesByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, rules.AllowedOrders)
}

type listRulesByGroupReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listRulesByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, rules.AllowedOrders)
}

type updateRuleReq struct {
	token string
	id    string
	rule
}

func (req updateRuleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingRuleID
	}

	return req.rule.validate()
}

type removeRulesReq struct {
	token   string
	RuleIDs []string `json:"rule_ids"`
}

func (req removeRulesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.RuleIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, ruleID := range req.RuleIDs {
		if ruleID == "" {
			return apiutil.ErrMissingRuleID
		}
	}

	return nil
}

type thingRulesReq struct {
	token   string
	thingID string
	RuleIDs []string `json:"rule_ids"`
}

func (req thingRulesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.RuleIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, ruleID := range req.RuleIDs {
		if ruleID == "" {
			return apiutil.ErrMissingRuleID
		}
	}

	return nil
}
