// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"slices"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/gofrs/uuid"
)

const (
	minLen        = 1
	maxLimitSize  = 200
	maxNameSize   = 254
	maxThingIDs   = 100
	minAlarmLevel = 1
	maxAlarmLevel = 5
)

type rule struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Input       rules.Input       `json:"input"`
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
		if len(rule.Input.ThingIDs) < minLen || len(rule.Input.ThingIDs) > maxThingIDs {
			return apiutil.ErrThingIDsSize
		}
		for _, id := range rule.Input.ThingIDs {
			if _, err := uuid.FromString(id); err != nil {
				return apiutil.ErrInvalidIDFormat
			}
		}
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

	switch req.Input.Type {
	case rules.InputTypeMessage, rules.InputTypeAlarm, rules.InputTypeSchedule, rules.InputTypeCommand:
	default:
		return apiutil.ErrInvalidInputType
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
			if action.Level < minAlarmLevel || action.Level > maxAlarmLevel {
				return apiutil.ErrInvalidAlarmLevel
			}
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
	pageMetadata rules.PageMetadata
}

func (req listRulesByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return req.pageMetadata.Validate(maxLimitSize, maxNameSize)
}

type listRulesByGroupReq struct {
	token        string
	groupID      string
	pageMetadata rules.PageMetadata
}

func (req listRulesByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return req.pageMetadata.Validate(maxLimitSize, maxNameSize)
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

type ruleThingsReq struct {
	token    string
	ruleID   string
	ThingIDs []string `json:"thing_ids"`
}

func (req ruleThingsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ruleID == "" {
		return apiutil.ErrMissingRuleID
	}

	if len(req.ThingIDs) < minLen || len(req.ThingIDs) > maxThingIDs {
		return apiutil.ErrThingIDsSize
	}

	for _, id := range req.ThingIDs {
		if _, err := uuid.FromString(id); err != nil {
			return apiutil.ErrInvalidIDFormat
		}
	}

	return nil
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

	if slices.Contains(req.RuleIDs, "") {
		return apiutil.ErrMissingRuleID
	}

	return nil
}
