// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"
	"github.com/go-kit/kit/endpoint"
)

func createScriptsEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createScriptsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var scripts []pyscripts.Script
		for _, s := range req.Scripts {
			scripts = append(scripts, pyscripts.Script{
				Name:        s.Name,
				Description: s.Description,
				Source:      s.Source,
			})
		}

		saved, err := svc.CreateScripts(ctx, req.token, req.groupID, scripts...)
		if err != nil {
			return nil, err
		}

		return buildScriptsRes(saved, true), nil
	}
}

func listScriptsByGroupEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listScriptsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListScriptsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildScriptsPageRes(page, req.pageMetadata), nil
	}
}

func viewScriptEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(scriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script, err := svc.ViewScript(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildScriptRes(script, true), nil
	}
}

func updateScriptEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateScriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		script := pyscripts.Script{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Source:      req.Source,
		}

		if err := svc.UpdateScript(ctx, req.token, script); err != nil {
			return nil, err
		}

		return scriptRes{ID: req.id, updated: true}, nil
	}
}

func removeScriptsEndpoint(svc pyscripts.Service) endpoint.Endpoint {
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

func runScriptEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(runScriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		in := runner.Input{
			Payload:     req.Payload,
			Subtopic:    req.Subtopic,
			Created:     req.Created,
			PublisherID: req.PublisherID,
		}

		run, err := svc.RunScript(ctx, req.token, req.id, in)
		if err != nil {
			return nil, err
		}

		return buildRunRes(run), nil
	}
}

func listRunsByScriptEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listRunsByScriptReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListRunsByScript(ctx, req.token, req.scriptID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildRunsPageRes(page, req.pageMetadata), nil
	}
}

func removeRunsEndpoint(svc pyscripts.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeRunsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveRuns(ctx, req.token, req.RunIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}
