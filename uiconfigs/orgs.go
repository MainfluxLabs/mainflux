// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Config map[string]any

type OrgConfig struct {
	OrgID  string
	Config Config
}

type OrgConfigPage struct {
	Total       uint64
	OrgsConfigs []OrgConfig
}

type OrgConfigBackup struct {
	OrgsConfigs []OrgConfig
}

type OrgConfigRepository interface {
	// Save creates a new organization config record in the database.
	Save(ctx context.Context, oc OrgConfig) (OrgConfig, error)

	// RetrieveByOrg returns the organization config associated with the given org ID
	RetrieveByOrg(ctx context.Context, orgID string) (OrgConfig, error)

	// RetrieveAll retrieves all organization configs.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (OrgConfigPage, error)

	// Update performs an update to the existing organization config.
	Update(ctx context.Context, oc OrgConfig) (OrgConfig, error)

	// Remove removes the organization configs with the provided identifier.
	Remove(ctx context.Context, orgID string) error

	// BackupAll retrieves all org configs.
	BackupAll(ctx context.Context) (OrgConfigBackup, error)
}
