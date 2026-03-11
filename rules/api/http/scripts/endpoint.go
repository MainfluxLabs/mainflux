// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package scripts

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
)

func createScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var reqScripts []rules.LuaScript
		for _, sReq := range req.Scripts {
			script := rules.LuaScript{
				Name:        sReq.Name,
				Script:      sReq.Script,
				Description: sReq.Description,
			}

			reqScripts = append(reqScripts, script)
		}

		scripts, err := svc.CreateScripts(ctx, req.token, req.groupID, reqScripts...)
		if err != nil {
			return nil, err
		}

		res := buildScriptsResponse(scripts, true)
		return res, nil
	}
}

func listScriptsByThingEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptsPageResponse(page, req.pageMetadata), nil
	}
}

func listScriptsByGroupEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptsPageResponse(page, req.pageMetadata), nil
	}
}

func listThingIDsByScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(scriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		ids, err := svc.ListThingIDsByScript(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := thingIDsRes{ThingIDs: ids}
		return res, nil
	}
}

func viewScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(scriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script, err := svc.ViewScript(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildScriptResponse(script, false), nil
	}
}

func updateScriptEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateScriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script := rules.LuaScript{
			ID:          req.id,
			Name:        req.Name,
			Script:      req.Script,
			Description: req.Description,
		}

		if err := svc.UpdateScript(ctx, req.token, script); err != nil {
			return nil, err
		}

		return scriptRes{updated: true}, nil
	}
}

func removeScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveScripts(ctx, req.token, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func assignScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignScripts(ctx, req.token, req.thingID, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return thingScriptsRes{}, nil
	}
}

func unassignScriptsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignScripts(ctx, req.token, req.thingID, req.ScriptIDs...); err != nil {
			return nil, err
		}

		return thingScriptsRes{}, nil
	}
}

func listScriptRunsByThingEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptRunsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptRunsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptRunsPageResponse(page, req.pageMetadata), nil
	}
}

func removeScriptRunsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeScriptRunsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveScriptRuns(ctx, req.token, req.ScriptRunIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildScriptsResponse(scripts []rules.LuaScript, created bool) scriptsRes {
	res := scriptsRes{Scripts: []scriptRes{}, created: created}

	for _, s := range scripts {
		sr := scriptRes{
			ID:          s.ID,
			GroupID:     s.GroupID,
			Name:        s.Name,
			Script:      s.Script,
			Description: s.Description,
		}
		res.Scripts = append(res.Scripts, sr)
	}

	return res
}

func buildScriptsPageResponse(page rules.LuaScriptsPage, pm apiutil.PageMetadata) scriptsPageRes {
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
		sr := scriptRes{
			ID:          s.ID,
			GroupID:     s.GroupID,
			Name:        s.Name,
			Script:      s.Script,
			Description: s.Description,
		}
		res.Scripts = append(res.Scripts, sr)
	}

	return res
}

func buildScriptResponse(s rules.LuaScript, updated bool) scriptRes {
	return scriptRes{
		ID:          s.ID,
		GroupID:     s.GroupID,
		Name:        s.Name,
		Script:      s.Script,
		Description: s.Description,
		updated:     updated,
	}
}

func buildScriptRunsPageResponse(page rules.ScriptRunsPage, pm apiutil.PageMetadata) scriptRunsPageRes {
	res := scriptRunsPageRes{
		pageRes: pageRes{
			Total:  page.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Runs: []scriptRunRes{},
	}

	for _, run := range page.Runs {
		sr := scriptRunRes{
			ID:         run.ID,
			ScriptID:   run.ScriptID,
			ThingID:    run.ThingID,
			Logs:       run.Logs,
			StartedAt:  run.StartedAt,
			FinishedAt: run.FinishedAt,
			Status:     run.Status,
			Error:      run.Error,
		}
		res.Runs = append(res.Runs, sr)
	}

	return res
}
