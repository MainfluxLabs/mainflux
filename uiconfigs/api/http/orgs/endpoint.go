// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package orgs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/go-kit/kit/endpoint"
)

func viewOrgConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewOrgConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		orgConfig, err := svc.ViewOrgConfig(ctx, req.token, req.orgID)
		if err != nil {
			return nil, err
		}

		res := buildOrgConfigResponse(orgConfig)
		return res, nil
	}
}

func listOrgsConfigsEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgsConfigsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgsConfigs(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildOrgsConfigsResponse(page, req.pageMetadata), nil
	}
}

func updateOrgConfigEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateOrgConfigReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		orgConfig := uiconfigs.OrgConfig{
			OrgID:  req.orgID,
			Config: req.Config,
		}

		_, err := svc.UpdateOrgConfig(ctx, req.token, orgConfig)
		if err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func buildOrgConfigResponse(oc uiconfigs.OrgConfig) OrgConfigResponse {
	return OrgConfigResponse{
		OrgID:  oc.OrgID,
		Config: oc.Config,
	}
}

func buildOrgsConfigsResponse(page uiconfigs.OrgConfigPage, pm apiutil.PageMetadata) orgsConfigsRes {
	res := orgsConfigsRes{
		OrgsConfigs: []OrgConfigResponse{},
		Total:       page.Total,
		Offset:      pm.Offset,
		Limit:       pm.Limit,
	}

	for _, oc := range page.OrgsConfigs {
		res.OrgsConfigs = append(res.OrgsConfigs, buildOrgConfigResponse(oc))
	}
	return res
}
