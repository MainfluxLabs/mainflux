package filestore

import (
	"context"
)

// FileThingsPage contains page related metadata as well as list of files that
// belong to this page.
type FileThingsPage struct {
	PageMetadata
	Files []FileInfo
}

// ThingsRepository represents filestore interface.
type ThingsRepository interface {
	// Save stores path in filestore
	Save(ctx context.Context, thingID, groupID string, fi FileInfo) error
	// Update updates path in filestore
	Update(ctx context.Context, thingID string, fi FileInfo) error
	// Retrieve retrieves path from filestore
	Retrieve(ctx context.Context, thingID string, fi FileInfo) (FileInfo, error)
	// RetrieveByThing retrieves files from filestore by thing
	RetrieveByThing(ctx context.Context, thingID string, fi FileInfo, pm PageMetadata) (FileThingsPage, error)
	// Remove removes path from filestore
	Remove(ctx context.Context, thingID string, fi FileInfo) error
	// RemoveByThing removes paths from filestore by thing ID
	RemoveByThing(ctx context.Context, thingID string) error
	// RemoveByGroup removes paths from filestore by group ID
	RemoveByGroup(ctx context.Context, groupID string) error

	// RetrieveThingIDsByGroup retrieves thing IDs from filestore by group ID
	RetrieveThingIDsByGroup(ctx context.Context, groupID string) ([]string, error)
}
