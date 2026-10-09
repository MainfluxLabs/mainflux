// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package pyscripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateScripts persists multiple Python scripts.
	CreateScripts(ctx context.Context, token, groupID string, scripts ...Script) ([]Script, error)

	// ListScriptsByGroup retrieves a list of scripts belonging to a specific Group.
	ListScriptsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ScriptsPage, error)

	// ViewScript retrieves a specific Script by its ID.
	ViewScript(ctx context.Context, token, id string) (Script, error)

	// UpdateScript updates the Script identified by the provided ID.
	UpdateScript(ctx context.Context, token string, script Script) error

	// RemoveScripts removes the Scripts identified by the provided IDs.
	RemoveScripts(ctx context.Context, token string, ids ...string) error

	// RemoveScriptsByGroup removes all Scripts belonging to a specific Group.
	RemoveScriptsByGroup(ctx context.Context, groupID string) error

	// RunScript executes the Script identified by the provided ID and records the run.
	RunScript(ctx context.Context, token, id string, in runner.Input) (ScriptRun, error)

	// ListRunsByScript retrieves a list of Runs belonging to a specific Script.
	ListRunsByScript(ctx context.Context, token, scriptID string, pm PageMetadata) (ScriptRunsPage, error)

	// RemoveRuns removes the Runs identified by the provided IDs.
	RemoveRuns(ctx context.Context, token string, ids ...string) error
}

type pyscriptsService struct {
	scripts    ScriptRepository
	runner     runner.Runner
	things     domain.ThingsClient
	idProvider uuid.IDProvider
	logger     logger.Logger
}

var _ Service = (*pyscriptsService)(nil)

// New instantiates the pyscripts service implementation.
func New(scripts ScriptRepository, run runner.Runner, things domain.ThingsClient, idp uuid.IDProvider, logger logger.Logger) Service {
	return &pyscriptsService{
		scripts:    scripts,
		runner:     run,
		things:     things,
		idProvider: idp,
		logger:     logger,
	}
}

func (ps *pyscriptsService) CreateScripts(ctx context.Context, token, groupID string, scripts ...Script) ([]Script, error) {
	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupEditor}); err != nil {
		return []Script{}, err
	}

	for i := range scripts {
		if err := ps.validateSource(ctx, scripts[i].Source); err != nil {
			return []Script{}, err
		}

		id, err := ps.idProvider.ID()
		if err != nil {
			return []Script{}, err
		}

		scripts[i].ID = id
		scripts[i].GroupID = groupID
		scripts[i].SHA256 = Checksum(scripts[i].Source)
	}

	return ps.scripts.SaveScripts(ctx, scripts...)
}

func (ps *pyscriptsService) ListScriptsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ScriptsPage, error) {
	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return ScriptsPage{}, err
	}

	return ps.scripts.RetrieveScriptsByGroup(ctx, groupID, pm)
}

func (ps *pyscriptsService) ViewScript(ctx context.Context, token, id string) (Script, error) {
	script, err := ps.scripts.RetrieveScriptByID(ctx, id)
	if err != nil {
		return Script{}, err
	}

	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: script.GroupID, Action: domain.GroupViewer}); err != nil {
		return Script{}, err
	}

	return script, nil
}

func (ps *pyscriptsService) UpdateScript(ctx context.Context, token string, script Script) error {
	existing, err := ps.scripts.RetrieveScriptByID(ctx, script.ID)
	if err != nil {
		return err
	}

	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: existing.GroupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	if err := ps.validateSource(ctx, script.Source); err != nil {
		return err
	}

	script.GroupID = existing.GroupID
	script.SHA256 = Checksum(script.Source)

	return ps.scripts.UpdateScript(ctx, script)
}

func (ps *pyscriptsService) RemoveScripts(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		script, err := ps.scripts.RetrieveScriptByID(ctx, id)
		if err != nil {
			return err
		}

		if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: script.GroupID, Action: domain.GroupEditor}); err != nil {
			return err
		}
	}

	return ps.scripts.RemoveScripts(ctx, ids...)
}

func (ps *pyscriptsService) RemoveScriptsByGroup(ctx context.Context, groupID string) error {
	return ps.scripts.RemoveScriptsByGroup(ctx, groupID)
}

func (ps *pyscriptsService) RunScript(ctx context.Context, token, id string, in runner.Input) (ScriptRun, error) {
	script, err := ps.scripts.RetrieveScriptByID(ctx, id)
	if err != nil {
		return ScriptRun{}, err
	}

	// Running a script executes code inside the service, so it is an edit-level
	// action rather than a read, even though it leaves the script unchanged.
	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: script.GroupID, Action: domain.GroupEditor}); err != nil {
		return ScriptRun{}, err
	}

	res, err := ps.runner.Run(ctx, runner.Source{SHA256: script.SHA256, Code: script.Source}, in)
	if err != nil {
		return ScriptRun{}, err
	}

	runID, err := ps.idProvider.ID()
	if err != nil {
		return ScriptRun{}, err
	}

	run := toRun(runID, script, res)

	// The script has already executed, so its record is persisted on a context
	// detached from the request: a client that disconnected mid-run would
	// otherwise cancel the write and lose the only trace of what ran.
	saved, err := ps.scripts.SaveRuns(context.WithoutCancel(ctx), run)
	if err != nil {
		return ScriptRun{}, err
	}

	return saved[0], nil
}

func (ps *pyscriptsService) ListRunsByScript(ctx context.Context, token, scriptID string, pm PageMetadata) (ScriptRunsPage, error) {
	script, err := ps.scripts.RetrieveScriptByID(ctx, scriptID)
	if err != nil {
		return ScriptRunsPage{}, err
	}

	if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: script.GroupID, Action: domain.GroupViewer}); err != nil {
		return ScriptRunsPage{}, err
	}

	return ps.scripts.RetrieveRunsByScript(ctx, scriptID, pm)
}

func (ps *pyscriptsService) RemoveRuns(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		run, err := ps.scripts.RetrieveRunByID(ctx, id)
		if err != nil {
			return err
		}

		script, err := ps.scripts.RetrieveScriptByID(ctx, run.ScriptID)
		if err != nil {
			return err
		}

		if err := ps.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: script.GroupID, Action: domain.GroupEditor}); err != nil {
			return err
		}
	}

	return ps.scripts.RemoveRuns(ctx, ids...)
}

// validateSource enforces the size cap and rejects source that does not compile.
// Both are enforced here rather than only at the transport layer, so every
// caller is bound by them.
func (ps *pyscriptsService) validateSource(ctx context.Context, source string) error {
	if len(source) > MaxScriptSize {
		return ErrScriptSize
	}

	msg, err := ps.runner.Validate(ctx, source)
	if err != nil {
		return err
	}

	if msg != "" {
		return errors.Wrap(ErrScriptSyntax, errors.New(msg))
	}

	return nil
}

// Checksum returns the hex-encoded SHA-256 of a script source.
func Checksum(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}

func toRun(id string, script Script, res runner.Result) ScriptRun {
	run := ScriptRun{
		ID:           id,
		ScriptID:     script.ID,
		ScriptSHA256: script.SHA256,
		Value:        res.Value,
		Logs:         res.Logs,
		StartedAt:    res.StartedAt,
		FinishedAt:   res.FinishedAt,
		Status:       ScriptRunStatusSuccess,
		Error:        res.Error,
	}

	if res.Error != "" {
		run.Status = ScriptRunStatusFail
	}

	return run
}
