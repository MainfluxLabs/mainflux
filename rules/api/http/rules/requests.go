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

type createRule struct {
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
	Rules   []createRule `json:"rules"`
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

	for _, r := range req.Rules {
		if err := r.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req createRule) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if err := validateInputType(req.Input.Type); err != nil {
		return err
	}

	if err := validateThingIDs(req.Input.ThingIDs); err != nil {
		return err
	}

	switch req.Input.Type {
	case rules.InputTypeAlarm:
		if err := validateInputConfig(req.Input.Config); err != nil {
			return err
		}
	default:
		if err := validateConditions(req.Conditions, req.Operator); err != nil {
			return err
		}
	}

	return validateActions(req.Input.Type, req.Actions)
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

type updateRuleInput struct {
	Type   string            `json:"type"`
	Config rules.InputConfig `json:"config,omitempty"`
}

type updateRuleReq struct {
	token       string
	id          string
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Input       updateRuleInput   `json:"input"`
	Conditions  []rules.Condition `json:"conditions"`
	Operator    string            `json:"operator,omitempty"`
	Actions     []rules.Action    `json:"actions"`
}

func (req updateRuleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingRuleID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if err := validateInputType(req.Input.Type); err != nil {
		return err
	}

	switch req.Input.Type {
	case rules.InputTypeAlarm:
		if err := validateInputConfig(req.Input.Config); err != nil {
			return err
		}
	default:
		if err := validateConditions(req.Conditions, req.Operator); err != nil {
			return err
		}
	}

	return validateActions(req.Input.Type, req.Actions)
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

	return validateThingIDs(req.ThingIDs)
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

func validateThingIDs(ids []string) error {
	if len(ids) < minLen || len(ids) > maxThingIDs {
		return apiutil.ErrThingIDsSize
	}
	for _, id := range ids {
		if _, err := uuid.FromString(id); err != nil {
			return apiutil.ErrInvalidIDFormat
		}
	}
	return nil
}

func validateInputType(inputType string) error {
	switch inputType {
	case rules.InputTypeMessage, rules.InputTypeAlarm, rules.InputTypeCommand:
		return nil
	default:
		return apiutil.ErrInvalidInputType
	}
}

func validateInputConfig(config rules.InputConfig) error {
	if comp, ok := config["level_comparator"]; ok {
		s, ok := comp.(string)
		if !ok {
			return apiutil.ErrInvalidConditionComparator
		}
		switch s {
		case rules.ComparatorEQ, rules.ComparatorGT, rules.ComparatorLT, rules.ComparatorGTE, rules.ComparatorLTE:
		default:
			return apiutil.ErrInvalidConditionComparator
		}
	}

	return nil
}

func validateConditions(conditions []rules.Condition, operator string) error {
	if len(conditions) < minLen {
		return apiutil.ErrEmptyList
	}
	for _, condition := range conditions {
		if condition.Field == "" {
			return apiutil.ErrMissingConditionField
		}
		switch condition.Comparator {
		case rules.ComparatorEQ, rules.ComparatorGT, rules.ComparatorLT, rules.ComparatorGTE, rules.ComparatorLTE:
		default:
			return apiutil.ErrInvalidConditionComparator
		}
		if condition.Threshold == nil {
			return apiutil.ErrMissingConditionThreshold
		}
	}
	if len(conditions) > minLen {
		if operator != rules.OperatorAND && operator != rules.OperatorOR {
			return apiutil.ErrInvalidOperator
		}
	}
	return nil
}

func validateActions(inputType string, actions []rules.Action) error {
	if len(actions) < minLen {
		return apiutil.ErrEmptyList
	}
	for _, action := range actions {
		switch action.Type {
		case rules.ActionTypeSMTP, rules.ActionTypeSMPP:
			if action.ID == "" {
				return apiutil.ErrMissingActionID
			}
		case rules.ActionTypeAlarm:
			if inputType == rules.InputTypeAlarm {
				return apiutil.ErrInvalidActionType
			}
			if action.Level < minAlarmLevel || action.Level > maxAlarmLevel {
				return apiutil.ErrInvalidAlarmLevel
			}
		case rules.ActionTypeWebhook:
		default:
			return apiutil.ErrInvalidActionType
		}
	}
	return nil
}
