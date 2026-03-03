package filestore

import (
	"context"
)

// FileGroupsPage contains page related metadata as well as list of files that
// belong to this page.
type FileGroupsPage struct {
	PageMetadata
	Files []FileInfo
}

// GroupsRepository represents filestore interface.
type GroupsRepository interface {
	// Save stores path in filestore
	Save(ctx context.Context, groupID string, fi FileInfo) error
	// Update updates path in filestore
	Update(ctx context.Context, groupID string, fi FileInfo) error
	// Retrieve retrieves path from filestore
	Retrieve(ctx context.Context, groupID string, fi FileInfo) (FileInfo, error)
	// RetrieveByGroup retrieves files from filestore by group
	RetrieveByGroup(ctx context.Context, groupID string, fi FileInfo, pm PageMetadata) (FileGroupsPage, error)
	// Remove removes path from filestore
	Remove(ctx context.Context, groupID string, fi FileInfo) error
	// RemoveByGroup removes all paths from filestore by group ID
	RemoveByGroup(ctx context.Context, groupID string) error
}
