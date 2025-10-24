package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

var (
	_ apiutil.Response = (*removeRes)(nil)
	_ apiutil.Response = (*ruleResponse)(nil)
	_ apiutil.Response = (*rulesRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Ord    string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	Name   string `json:"name,omitempty"`
}

type ruleResponse struct {
	ID          string            `json:"id"`
	GroupID     string            `json:"group_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Conditions  []rules.Condition `json:"conditions"`
	Operator    string            `json:"operator"`
	Actions     []rules.Action    `json:"actions"`
	updated     bool
}

func (res ruleResponse) Code() int {
	return http.StatusOK
}

func (res ruleResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res ruleResponse) Empty() bool {
	return res.updated
}

type rulesRes struct {
	Rules   []ruleResponse `json:"rules"`
	created bool
}

func (res rulesRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res rulesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res rulesRes) Empty() bool {
	return false
}

type RulesPageRes struct {
	pageRes
	Rules []ruleResponse `json:"rules"`
}

func (res RulesPageRes) Code() int {
	return http.StatusOK
}

func (res RulesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res RulesPageRes) Empty() bool {
	return false
}

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}
