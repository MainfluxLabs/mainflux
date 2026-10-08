// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"
	"github.com/go-kit/kit/metrics"
)

var _ pyscripts.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     pyscripts.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc pyscripts.Service, counter metrics.Counter, latency metrics.Histogram) pyscripts.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateScripts(ctx context.Context, token, groupID string, scripts ...pyscripts.Script) (saved []pyscripts.Script, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_scripts").Add(1)
		ms.latency.With("method", "create_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateScripts(ctx, token, groupID, scripts...)
}

func (ms *metricsMiddleware) ListScriptsByGroup(ctx context.Context, token, groupID string, pm pyscripts.PageMetadata) (_ pyscripts.ScriptsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_scripts_by_group").Add(1)
		ms.latency.With("method", "list_scripts_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListScriptsByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewScript(ctx context.Context, token, id string) (_ pyscripts.Script, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_script").Add(1)
		ms.latency.With("method", "view_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewScript(ctx, token, id)
}

func (ms *metricsMiddleware) UpdateScript(ctx context.Context, token string, script pyscripts.Script) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_script").Add(1)
		ms.latency.With("method", "update_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateScript(ctx, token, script)
}

func (ms *metricsMiddleware) RemoveScripts(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_scripts").Add(1)
		ms.latency.With("method", "remove_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveScripts(ctx, token, ids...)
}

func (ms *metricsMiddleware) RemoveScriptsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_scripts_by_group").Add(1)
		ms.latency.With("method", "remove_scripts_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveScriptsByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) RunScript(ctx context.Context, token, id string, in runner.Input) (_ pyscripts.ScriptRun, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "run_script").Add(1)
		ms.latency.With("method", "run_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RunScript(ctx, token, id, in)
}

func (ms *metricsMiddleware) ListRunsByScript(ctx context.Context, token, scriptID string, pm pyscripts.PageMetadata) (_ pyscripts.ScriptRunsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_runs_by_script").Add(1)
		ms.latency.With("method", "list_runs_by_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRunsByScript(ctx, token, scriptID, pm)
}

func (ms *metricsMiddleware) RemoveRuns(ctx context.Context, token string, ids ...string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_runs").Add(1)
		ms.latency.With("method", "remove_runs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveRuns(ctx, token, ids...)
}
