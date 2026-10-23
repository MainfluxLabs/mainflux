// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/opentracing/opentracing-go"
)

const (
	saveScripts            = "save_scripts"
	retrieveScriptByID     = "retrieve_script_by_id"
	retrieveScriptsByGroup = "retrieve_scripts_by_group"
	updateScript           = "update_script"
	removeScripts          = "remove_scripts"
	removeScriptsByGroup   = "remove_scripts_by_group"
	saveRuns               = "save_runs"
	retrieveRunByID        = "retrieve_run_by_id"
	retrieveRunsByScript   = "retrieve_runs_by_script"
	removeRuns             = "remove_runs"
)

var _ pyscripts.ScriptRepository = (*scriptRepositoryMiddleware)(nil)

type scriptRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   pyscripts.ScriptRepository
}

// ScriptRepositoryMiddleware tracks request and their latency, and adds spans to context.
func ScriptRepositoryMiddleware(tracer opentracing.Tracer, repo pyscripts.ScriptRepository) pyscripts.ScriptRepository {
	return scriptRepositoryMiddleware{tracer: tracer, repo: repo}
}

func (srm scriptRepositoryMiddleware) SaveScripts(ctx context.Context, scripts ...pyscripts.Script) ([]pyscripts.Script, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, saveScripts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.SaveScripts(ctx, scripts...)
}

func (srm scriptRepositoryMiddleware) RetrieveScriptByID(ctx context.Context, id string) (pyscripts.Script, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveScriptByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveScriptByID(ctx, id)
}

func (srm scriptRepositoryMiddleware) RetrieveScriptsByGroup(ctx context.Context, groupID string, pm pyscripts.PageMetadata) (pyscripts.ScriptsPage, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveScriptsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveScriptsByGroup(ctx, groupID, pm)
}

func (srm scriptRepositoryMiddleware) UpdateScript(ctx context.Context, script pyscripts.Script) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, updateScript)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.UpdateScript(ctx, script)
}

func (srm scriptRepositoryMiddleware) RemoveScripts(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeScripts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveScripts(ctx, ids...)
}

func (srm scriptRepositoryMiddleware) RemoveScriptsByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeScriptsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveScriptsByGroup(ctx, groupID)
}

func (srm scriptRepositoryMiddleware) SaveRuns(ctx context.Context, runs ...pyscripts.ScriptRun) ([]pyscripts.ScriptRun, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, saveRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.SaveRuns(ctx, runs...)
}

func (srm scriptRepositoryMiddleware) RetrieveRunByID(ctx context.Context, id string) (pyscripts.ScriptRun, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveRunByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveRunByID(ctx, id)
}

func (srm scriptRepositoryMiddleware) RetrieveRunsByScript(ctx context.Context, scriptID string, pm pyscripts.PageMetadata) (pyscripts.ScriptRunsPage, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveRunsByScript)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveRunsByScript(ctx, scriptID, pm)
}

func (srm scriptRepositoryMiddleware) RemoveRuns(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveRuns(ctx, ids...)
}
