// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/go-kit/kit/endpoint"
)

func viewGroupConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewGroupConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupConfig, err := svc.ViewGroupConfig(ctx, req.token, req.groupID)
		if err != nil {
			return nil, err
		}

		res := buildGroupConfigResponse(groupConfig)
		return res, nil
	}
}

func listGroupsConfigsEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupsConfigsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroupsConfigs(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsConfigsResponse(page, req.pageMetadata), nil
	}
}

func updateGroupConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateGroupConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupConfig := uiconfigs.GroupConfig{
			GroupID: req.groupID,
			Config:  req.Config,
		}

		_, err := svc.UpdateGroupConfig(ctx, req.token, groupConfig)
		if err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusOK}, nil
	}
}

func buildGroupConfigResponse(gc uiconfigs.GroupConfig) GroupConfigResponse {
	return GroupConfigResponse{
		GroupID: gc.GroupID,
		Config:  gc.Config,
	}
}

func buildGroupsConfigsResponse(page uiconfigs.GroupConfigPage, pm apiutil.PageMetadata) groupsConfigsRes {
	res := groupsConfigsRes{
		GroupsConfigs: []GroupConfigResponse{},
		Total:         page.Total,
		Offset:        pm.Offset,
		Limit:         pm.Limit,
	}

	for _, gc := range page.GroupsConfigs {
		res.GroupsConfigs = append(res.GroupsConfigs, buildGroupConfigResponse(gc))
	}
	return res
}
