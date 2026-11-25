// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"fmt"
	"time"
)

const (
	RuleRunStatusSuccess = "success"
	RuleRunStatusFail    = "fail"
)

// RuleRun represents an execution of a rule that fired or failed.
type RuleRun struct {
	ID         string
	RuleID     string
	ThingID    string
	StartedAt  time.Time
	FinishedAt time.Time
	Status     string
	Details    RuleRunDetails
}

// RuleRunDetails represents the condition and action results of a rule run.
type RuleRunDetails struct {
	// Error describes a failure that belongs to no single condition or action.
	Error      string             `json:"error,omitempty"`
	Conditions []RuleRunCondition `json:"conditions,omitempty"`
	Actions    []RuleRunAction    `json:"actions,omitempty"`
}

// RuleRunCondition represents the outcome of a rule condition evaluation.
type RuleRunCondition struct {
	Type     string   `json:"type"`
	ScriptID string   `json:"script_id,omitempty"`
	Met      bool     `json:"met"`
	Result   any      `json:"result,omitempty"`
	Logs     []string `json:"logs,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// RuleRunAction represents the outcome of a rule action execution.
type RuleRunAction struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

func (d RuleRunDetails) failed() bool {
	if d.Error != "" {
		return true
	}
	for _, c := range d.Conditions {
		if c.Error != "" {
			return true
		}
	}
	for _, a := range d.Actions {
		if a.Error != "" {
			return true
		}
	}
	return false
}

func (rs *rulesService) saveRun(ctx context.Context, ruleID, thingID string, startedAt time.Time, details RuleRunDetails) {
	id, idErr := rs.idProvider.ID()
	if idErr != nil {
		rs.logger.Error(fmt.Sprintf("saving run of rule with id %s failed with error: %v", ruleID, idErr))
		return
	}

	run := RuleRun{
		ID:         id,
		RuleID:     ruleID,
		ThingID:    thingID,
		StartedAt:  startedAt,
		FinishedAt: time.Now(),
		Status:     RuleRunStatusSuccess,
		Details:    details,
	}
	if details.failed() {
		run.Status = RuleRunStatusFail
	}

	if _, err := rs.rules.SaveRuns(ctx, run); err != nil {
		rs.logger.Error(fmt.Sprintf("saving run of rule with id %s failed with error: %v", ruleID, err))
	}
}
