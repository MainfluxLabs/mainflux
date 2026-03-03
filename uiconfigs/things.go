// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type ThingConfig struct {
	ThingID string
	GroupID string
	Config  Config
}

type ThingConfigPage struct {
	Total         uint64
	ThingsConfigs []ThingConfig
}

type ThingConfigBackup struct {
	ThingsConfigs []ThingConfig
}

type ThingConfigRepository interface {
	// Save creates a new thing config record in the database.
	Save(ctx context.Context, tc ThingConfig) (ThingConfig, error)

	// RetrieveByThing returns the thing config associated with the given thing ID.
	RetrieveByThing(ctx context.Context, thingID string) (ThingConfig, error)

	// RetrieveAll retrieves all thing configs.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (ThingConfigPage, error)

	// Update performs an update to the existing thing config.
	Update(ctx context.Context, tc ThingConfig) (ThingConfig, error)

	// Remove removes the thing configs with the given thing ID.
	Remove(ctx context.Context, thingID string) error

	// RemoveByGroup deletes all thing configs that belong to the given group ID
	RemoveByGroup(ctx context.Context, groupID string) error

	// BackupAll retrieves all thing configs.
	BackupAll(ctx context.Context) (ThingConfigBackup, error)
}
