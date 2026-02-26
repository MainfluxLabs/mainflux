package http

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

var (
	_ apiutil.Response = (*removeRes)(nil)
	_ apiutil.Response = (*ruleResponse)(nil)
	_ apiutil.Response = (*rulesRes)(nil)
	_ apiutil.Response = (*thingIDsRes)(nil)
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

type thingIDsRes struct {
	ThingIDs []string `json:"thing_ids"`
}

func (res thingIDsRes) Code() int {
	return http.StatusOK
}

func (res thingIDsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingIDsRes) Empty() bool {
	return false
}

type thingRulesRes struct{}

func (res thingRulesRes) Code() int {
	return http.StatusOK
}

func (res thingRulesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingRulesRes) Empty() bool {
	return true
}

type scriptRes struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Name        string `json:"name"`
	Script      string `json:"script,omitempty"`
	Description string `json:"description,omitempty"`
	updated     bool
}

func (res scriptRes) Code() int {
	return http.StatusOK
}

func (res scriptRes) Headers() map[string]string {
	return map[string]string{}
}

func (res scriptRes) Empty() bool {
	return res.updated
}

type scriptsRes struct {
	Scripts []scriptRes `json:"scripts"`
	created bool
}

func (res scriptsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res scriptsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res scriptsRes) Empty() bool {
	return false
}

type scriptsPageRes struct {
	pageRes
	Scripts []scriptRes `json:"scripts"`
}

func (res scriptsPageRes) Code() int {
	return http.StatusOK
}

func (res scriptsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res scriptsPageRes) Empty() bool {
	return false
}

type scriptRunRes struct {
	ID         string    `json:"id"`
	ScriptID   string    `json:"script_id"`
	ThingID    string    `json:"thing_id"`
	Logs       []string  `json:"logs"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
}

func (res scriptRunRes) Code() int {
	return http.StatusOK
}

func (res scriptRunRes) Headers() map[string]string {
	return map[string]string{}
}

func (res scriptRunRes) Empty() bool {
	return false
}

type scriptRunsPageRes struct {
	pageRes
	Runs []scriptRunRes `json:"runs"`
}

func (res scriptRunsPageRes) Code() int {
	return http.StatusOK
}

func (res scriptRunsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res scriptRunsPageRes) Empty() bool {
	return false
}
