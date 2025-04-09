// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/rules"
)

var (
	ErrInvalidConditionField    = errors.New("missing or invalid condition field")
	ErrInvalidConditionOperator = errors.New("missing or invalid condition operator")
	ErrInvalidActionType        = errors.New("missing or invalid action type")
)

const (
	minLen       = 1
	maxLimitSize = 100
	maxNameSize  = 254
)

type ruleReq struct {
	Name        string          `json:"name"`
	Condition   rules.Condition `json:"condition"`
	Actions     []rules.Action  `json:"actions"`
	Description string          `json:"description,omitempty"`
}

type createRulesReq struct {
	token     string
	profileID string
	Rules     []ruleReq `json:"rules"`
}

func (req createRulesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.profileID == "" {
		return apiutil.ErrMissingProfileID
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

func (req ruleReq) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.Condition.Field == "" {
		return ErrInvalidConditionField
	}

	if req.Condition.Operator == "" {
		return ErrInvalidConditionOperator
	}

	//TODO: ADD Condition.Threshold validation

	if len(req.Actions) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, action := range req.Actions {
		if action.Type == "" {
			return ErrInvalidActionType
		}
	}

	return nil
}

type viewRuleReq struct {
	token string
	id    string
}

func (req *viewRuleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingRuleID
	}
	return nil
}

type listRulesByProfileReq struct {
	token        string
	profileID    string
	pageMetadata apiutil.PageMetadata
}

func (req listRulesByProfileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.profileID == "" {
		return apiutil.ErrMissingProfileID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
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

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type updateRuleReq struct {
	token string
	id    string
	ruleReq
}

func (req updateRuleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingRuleID
	}

	return req.ruleReq.validate()
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
