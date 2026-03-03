// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	orgs "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/orgs"
	things "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/things"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		fileName := fmt.Sprintf("uiconfigs-backup.json")
		return buildBackupResponse(backup, fileName)
	}
}

func restoreEndpoint(svc uiconfigs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup := buildBackup(req)

		if err := svc.Restore(ctx, req.token, backup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(b uiconfigs.Backup, fileName string) (apiutil.ViewFileRes, error) {
	views := backupResponse{
		OrgsConfigs:   make([]orgs.OrgConfigResponse, 0, len(b.OrgsConfigs)),
		ThingsConfigs: make([]things.ThingConfigResponse, 0, len(b.ThingsConfigs)),
	}

	for _, oc := range b.OrgsConfigs {
		views.OrgsConfigs = append(views.OrgsConfigs, orgs.OrgConfigResponse{
			OrgID:  oc.OrgID,
			Config: oc.Config,
		})
	}

	for _, tc := range b.ThingsConfigs {
		views.ThingsConfigs = append(views.ThingsConfigs, things.ThingConfigResponse{
			ThingID: tc.ThingID,
			GroupID: tc.GroupID,
			Config:  tc.Config,
		})
	}

	data, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		return apiutil.ViewFileRes{}, err
	}

	return apiutil.ViewFileRes{
		File:     data,
		FileName: fileName,
	}, nil
}

func buildBackup(req restoreReq) (backup uiconfigs.Backup) {
	for _, oc := range req.OrgsConfigs {
		backup.OrgsConfigs = append(backup.OrgsConfigs, uiconfigs.OrgConfig{
			OrgID:  oc.OrgID,
			Config: oc.Config,
		})
	}

	for _, tc := range req.ThingsConfigs {
		backup.ThingsConfigs = append(backup.ThingsConfigs, uiconfigs.ThingConfig{
			ThingID: tc.ThingID,
			GroupID: tc.GroupID,
			Config:  tc.Config,
		})
	}

	return backup
}
