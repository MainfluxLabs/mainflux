// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type GroupConfig struct {
	GroupID string
	Config  Config
}

type GroupConfigPage struct {
	Total         uint64
	GroupsConfigs []GroupConfig
}

type GroupConfigBackup struct {
	GroupsConfigs []GroupConfig
}

type GroupConfigRepository interface {
	// Save creates a new group config record in the database.
	Save(ctx context.Context, gc GroupConfig) (GroupConfig, error)

	// RetrieveByGroup returns the group config associated with the given group ID
	RetrieveByGroup(ctx context.Context, groupID string) (GroupConfig, error)

	// RetrieveAll retrieves all group configs.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (GroupConfigPage, error)

	// Update performs an update to the existing group config.
	Update(ctx context.Context, gc GroupConfig) (GroupConfig, error)

	// Remove removes the group configs with the provided identifier.
	Remove(ctx context.Context, groupID string) error

	// BackupAll retrieves all group configs.
	BackupAll(ctx context.Context) (GroupConfigBackup, error)
}
