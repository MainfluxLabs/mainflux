// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pyscripts"
)

var (
	_ apiutil.Response = (*scriptRes)(nil)
	_ apiutil.Response = (*scriptsRes)(nil)
	_ apiutil.Response = (*scriptsPageRes)(nil)
	_ apiutil.Response = (*runRes)(nil)
	_ apiutil.Response = (*runsPageRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Ord    string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	Name   string `json:"name,omitempty"`
}

type scriptRes struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Source      string    `json:"source,omitempty"`
	SHA256      string    `json:"sha256,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
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

type runRes struct {
	ID           string    `json:"id"`
	ScriptID     string    `json:"script_id"`
	ScriptSHA256 string    `json:"script_sha256"`
	Status       string    `json:"status"`
	Value        any       `json:"return_value"`
	Logs         []string  `json:"logs"`
	Error        string    `json:"error,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
}

func (res runRes) Code() int {
	return http.StatusOK
}

func (res runRes) Headers() map[string]string {
	return map[string]string{}
}

func (res runRes) Empty() bool {
	return false
}

type runsPageRes struct {
	pageRes
	Runs []runRes `json:"runs"`
}

func (res runsPageRes) Code() int {
	return http.StatusOK
}

func (res runsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res runsPageRes) Empty() bool {
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

func buildScriptRes(s pyscripts.Script, withSource bool) scriptRes {
	res := scriptRes{
		ID:          s.ID,
		GroupID:     s.GroupID,
		Name:        s.Name,
		Description: s.Description,
		SHA256:      s.SHA256,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}

	if withSource {
		res.Source = s.Source
	}

	return res
}

func buildScriptsRes(scripts []pyscripts.Script, created bool) scriptsRes {
	res := scriptsRes{created: created, Scripts: []scriptRes{}}
	for _, s := range scripts {
		res.Scripts = append(res.Scripts, buildScriptRes(s, false))
	}

	return res
}

func buildScriptsPageRes(page pyscripts.ScriptsPage, pm pyscripts.PageMetadata) scriptsPageRes {
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
		res.Scripts = append(res.Scripts, buildScriptRes(s, false))
	}

	return res
}

func buildRunRes(r pyscripts.ScriptRun) runRes {
	logs := r.Logs
	if logs == nil {
		logs = []string{}
	}

	return runRes{
		ID:           r.ID,
		ScriptID:     r.ScriptID,
		ScriptSHA256: r.ScriptSHA256,
		Status:       r.Status,
		Value:        r.Value,
		Logs:         logs,
		Error:        r.Error,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
	}
}

func buildRunsPageRes(page pyscripts.ScriptRunsPage, pm pyscripts.PageMetadata) runsPageRes {
	res := runsPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Ord:    pm.Order,
			Dir:    pm.Dir,
		},
		Runs: []runRes{},
	}

	for _, r := range page.Runs {
		res.Runs = append(res.Runs, buildRunRes(r))
	}

	return res
}
