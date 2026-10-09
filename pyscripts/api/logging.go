// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"

	log "github.com/MainfluxLabs/mainflux/logger"
)

var _ pyscripts.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    pyscripts.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc pyscripts.Service, logger log.Logger) pyscripts.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm loggingMiddleware) CreateScripts(ctx context.Context, token, groupID string, scripts ...pyscripts.Script) (saved []pyscripts.Script, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method create_scripts by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateScripts(ctx, token, groupID, scripts...)
}

func (lm loggingMiddleware) ListScriptsByGroup(ctx context.Context, token, groupID string, pm pyscripts.PageMetadata) (_ pyscripts.ScriptsPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_scripts_by_group by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListScriptsByGroup(ctx, token, groupID, pm)
}

func (lm loggingMiddleware) ViewScript(ctx context.Context, token, id string) (_ pyscripts.Script, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method view_script by user %s, script id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewScript(ctx, token, id)
}

func (lm loggingMiddleware) UpdateScript(ctx context.Context, token string, script pyscripts.Script) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method update_script by user %s, script id %s took %s to complete", email, script.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateScript(ctx, token, script)
}

func (lm loggingMiddleware) RemoveScripts(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_scripts by user %s, script ids %v took %s to complete", email, ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveScripts(ctx, token, ids...)
}

func (lm loggingMiddleware) RemoveScriptsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_scripts_by_group group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveScriptsByGroup(ctx, groupID)
}

func (lm loggingMiddleware) RunScript(ctx context.Context, token, id string, in runner.Input) (_ pyscripts.ScriptRun, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method run_script by user %s, script id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RunScript(ctx, token, id, in)
}

func (lm loggingMiddleware) ListRunsByScript(ctx context.Context, token, scriptID string, pm pyscripts.PageMetadata) (_ pyscripts.ScriptRunsPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_runs_by_script by user %s, script id %s took %s to complete", email, scriptID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRunsByScript(ctx, token, scriptID, pm)
}

func (lm loggingMiddleware) RemoveRuns(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method remove_runs by user %s, run ids %v took %s to complete", email, ids, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveRuns(ctx, token, ids...)
}
