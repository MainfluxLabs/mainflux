// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/go-kit/kit/endpoint"
)

func viewThingConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewThingConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		thingConfig, err := svc.ViewThingConfig(ctx, req.token, req.thingID)
		if err != nil {
			return nil, err
		}

		res := buildThingConfigResponse(thingConfig)
		return res, nil
	}
}

func listThingsConfigsEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listThingsConfigsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsConfigs(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsConfigsResponse(page, req.pageMetadata), nil
	}
}

func updateThingConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateThingConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		thingConfig := uiconfigs.ThingConfig{
			ThingID: req.thingID,
			Config:  req.Config,
		}

		_, err := svc.UpdateThingConfig(ctx, req.token, thingConfig)
		if err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func buildThingConfigResponse(tc uiconfigs.ThingConfig) ThingConfigResponse {
	tcb := ThingConfigResponse{
		ThingID: tc.ThingID,
		Config:  tc.Config,
	}

	return tcb
}

func buildThingsConfigsResponse(page uiconfigs.ThingConfigPage, pm apiutil.PageMetadata) thingsConfigsRes {
	res := thingsConfigsRes{
		ThingsConfigs: []ThingConfigResponse{},
		Total:         page.Total,
		Offset:        pm.Offset,
		Limit:         pm.Limit,
	}

	for _, tc := range page.ThingsConfigs {
		res.ThingsConfigs = append(res.ThingsConfigs, buildThingConfigResponse(tc))
	}
	return res
}
