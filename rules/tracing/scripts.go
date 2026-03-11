package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveScript                = "save_script"
	retrieveScriptByID        = "retrieve_script_by_id"
	retrieveScriptsByThing    = "retrieve_scripts_by_thing"
	retrieveScriptsByGroup    = "retrieve_scripts_by_group"
	retrieveThingIDsByScript  = "retrieve_thing_ids_by_script"
	updateScript              = "update_script"
	removeScripts             = "remove_scripts"
	removeScriptsByGroup      = "remove_scripts_by_group"
	assignScripts             = "assign_scripts"
	unassignScripts           = "unassign_scripts"
	unassignScriptsFromThing  = "unassign_scripts_from_thing"
	saveScriptRuns            = "save_script_runs"
	retrieveScriptRunsByThing = "retrieve_script_runs_by_thing"
	removeScriptRuns          = "remove_script_runs"
	retrieveScriptRunByID     = "retrieve_script_run_by_id"
)

func (rpm ruleRepositoryMiddleware) SaveScripts(ctx context.Context, scripts ...rules.LuaScript) ([]rules.LuaScript, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, saveScript)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.SaveScripts(ctx, scripts...)
}

func (rpm ruleRepositoryMiddleware) RetrieveScriptByID(ctx context.Context, id string) (rules.LuaScript, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveScriptByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveScriptByID(ctx, id)
}

func (rpm ruleRepositoryMiddleware) RetrieveScriptsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveScriptsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveScriptsByThing(ctx, thingID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveScriptsByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveScriptsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveScriptsByGroup(ctx, groupID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveThingIDsByScript(ctx context.Context, scriptID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveThingIDsByScript)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveThingIDsByScript(ctx, scriptID)
}

func (rpm ruleRepositoryMiddleware) UpdateScript(ctx context.Context, script rules.LuaScript) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, updateScript)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UpdateScript(ctx, script)
}

func (rpm ruleRepositoryMiddleware) RemoveScripts(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeScripts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RemoveScripts(ctx, ids...)
}

func (rpm ruleRepositoryMiddleware) RemoveScriptsByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeScriptsByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RemoveScriptsByGroup(ctx, groupID)
}

func (rpm ruleRepositoryMiddleware) AssignScripts(ctx context.Context, thingID string, scriptIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, assignScripts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.AssignScripts(ctx, thingID, scriptIDs...)
}

func (rpm ruleRepositoryMiddleware) UnassignScriptsFromThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignScriptsFromThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UnassignScriptsFromThing(ctx, thingID)
}

func (rpm ruleRepositoryMiddleware) UnassignScripts(ctx context.Context, thingID string, scriptIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignScripts)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UnassignScripts(ctx, thingID, scriptIDs...)
}

func (rpm ruleRepositoryMiddleware) SaveScriptRuns(ctx context.Context, runs ...rules.ScriptRun) ([]rules.ScriptRun, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, saveScriptRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.SaveScriptRuns(ctx, runs...)
}

func (rpm ruleRepositoryMiddleware) RetrieveScriptRunsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.ScriptRunsPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveScriptRunsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveScriptRunsByThing(ctx, thingID, pm)
}

func (rpm ruleRepositoryMiddleware) RemoveScriptRuns(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeScriptRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RemoveScriptRuns(ctx, ids...)
}

func (rpm ruleRepositoryMiddleware) RetrieveScriptRunByID(ctx context.Context, id string) (rules.ScriptRun, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveScriptRunByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveScriptRunByID(ctx, id)
}
