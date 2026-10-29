// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package pyscripts

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	ErrScriptSize   = errors.New("script size exceeds limit")
	ErrScriptSyntax = errors.New("script is not valid python")
	ErrRunnerBusy   = errors.New("no python worker available")
	ErrRunnerFailed = errors.New("python worker failed")
	ErrOutputLimit  = errors.New("output limit exceeded")
	ErrRunnerClosed = errors.New("runner is closed")
)

const (
	ScriptRunStatusSuccess = "success"
	ScriptRunStatusFail    = "fail"
)

type Script struct {
	ID          string
	GroupID     string
	Name        string
	Description string
	Source      string
	Checksum    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ScriptsPage struct {
	Total   uint64
	Scripts []Script
}

type ScriptRun struct {
	ID             string
	ScriptID       string
	ScriptChecksum string
	Value          any
	Logs           []string
	StartedAt      time.Time
	FinishedAt     time.Time
	Status         string
	Error          string
}

type ScriptRunsPage struct {
	Total uint64
	Runs  []ScriptRun
}

type PageMetadata struct {
	Total  uint64    `json:"total,omitempty"`
	Offset uint64    `json:"offset,omitempty"`
	Limit  uint64    `json:"limit,omitempty"`
	Order  string    `json:"order,omitempty"`
	Dir    string    `json:"dir,omitempty"`
	Name   string    `json:"name,omitempty"`
	Status string    `json:"status,omitempty"`
	From   time.Time `json:"from,omitempty"`
	To     time.Time `json:"to,omitempty"`
}

// ScriptOrderFields maps API-facing order keys to SQL column expressions for the scripts table.
var ScriptOrderFields = map[string]string{
	"id":         "id",
	"name":       "LOWER(name)",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

// ScriptRunOrderFields maps API-facing order keys to SQL column expressions for the script_runs table.
var ScriptRunOrderFields = map[string]string{
	"id":         "id",
	"started_at": "started_at",
	"status":     "status",
}

// Source is a script as the runner needs it: the checksum keys the runner's own
// cache, and the code is sent only when that cache may not hold it.
type Source struct {
	Checksum string
	Code     string
}

// Input is what a run is given. It is the message shape the Python side sees as
// its "message" global, so the json tags are part of the contract with it.
type Input struct {
	Payload     map[string]any `json:"payload"`
	Subtopic    string         `json:"subtopic,omitempty"`
	Created     int64          `json:"created,omitempty"`
	ThingID     string         `json:"thing_id,omitempty"`
}

// Result is the outcome of one run. Error is either the script's own failure or
// the reason the run was cut short; the two are not distinguished here because
// a caller records both the same way.
type Result struct {
	Value      any
	Logs       []string
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

type Runner interface {
	Run(ctx context.Context, src Source, in Input) (Result, error)

	// Validate compiles code without executing it. It returns the Python syntax
	// error as a string when the source does not compile, and an empty string
	// when it does. A non-nil error means the check itself could not be run.
	Validate(ctx context.Context, code string) (string, error)

	Close() error
}

type ScriptRepository interface {
	// SaveScripts persists multiple Python scripts.
	SaveScripts(ctx context.Context, scripts ...Script) ([]Script, error)

	// RetrieveScriptByID retrieves a single Script denoted by ID.
	RetrieveScriptByID(ctx context.Context, id string) (Script, error)

	// RetrieveScriptsByGroup retrieves a list of Scripts belonging to a specific Group.
	RetrieveScriptsByGroup(ctx context.Context, groupID string, pm PageMetadata) (ScriptsPage, error)

	// UpdateScript updates the Script denoted by script.ID.
	UpdateScript(ctx context.Context, script Script) error

	// RemoveScripts removes the Scripts denoted by the provided IDs.
	RemoveScripts(ctx context.Context, ids ...string) error

	// RemoveScriptsByGroup removes all Scripts belonging to a specific Group.
	RemoveScriptsByGroup(ctx context.Context, groupID string) error

	// SaveRuns persists multiple Script Runs.
	SaveRuns(ctx context.Context, runs ...ScriptRun) ([]ScriptRun, error)

	// RetrieveRunByID retrieves a single Run denoted by ID.
	RetrieveRunByID(ctx context.Context, id string) (ScriptRun, error)

	// RetrieveRunsByScript retrieves a list of Runs belonging to a specific Script.
	RetrieveRunsByScript(ctx context.Context, scriptID string, pm PageMetadata) (ScriptRunsPage, error)

	// RemoveRuns removes the Runs denoted by the provided IDs.
	RemoveRuns(ctx context.Context, ids ...string) error
}
