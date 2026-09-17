// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
)

const (
	statusKey = "status"
	fromKey   = "from"
	toKey     = "to"
)

type ScriptRunRes struct {
	ID         string    `json:"id"`
	ScriptID   string    `json:"script_id"`
	RuleID     string    `json:"rule_id"`
	ThingID    string    `json:"thing_id"`
	Logs       []string  `json:"logs"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
}

func BuildScriptRunsPageMetadata(r *http.Request) (rules.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return rules.PageMetadata{}, err
	}

	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return rules.PageMetadata{}, err
	}

	status, err := apiutil.ReadStringQuery(r, statusKey, "")
	if err != nil {
		return rules.PageMetadata{}, err
	}

	fromMs, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return rules.PageMetadata{}, err
	}

	toMs, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return rules.PageMetadata{}, err
	}

	var from, to time.Time
	if fromMs > 0 {
		from = time.UnixMilli(fromMs)
	}
	if toMs > 0 {
		to = time.UnixMilli(toMs)
	}

	return rules.PageMetadata{
		Offset: base.Offset,
		Limit:  base.Limit,
		Order:  base.Order,
		Dir:    base.Dir,
		Name:   name,
		Status: status,
		From:   from,
		To:     to,
	}, nil
}

func ToScriptRunRes(run rules.ScriptRun) ScriptRunRes {
	return ScriptRunRes{
		ID:         run.ID,
		ScriptID:   run.ScriptID,
		RuleID:     run.RuleID,
		ThingID:    run.ThingID,
		Logs:       run.Logs,
		StartedAt:  run.StartedAt,
		FinishedAt: run.FinishedAt,
		Status:     run.Status,
		Error:      run.Error,
	}
}
