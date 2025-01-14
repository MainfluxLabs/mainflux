// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
)

// Metadata to be used for Mainflux thing or profile for customized
// describing of particular thing or profile.
type Metadata map[string]interface{}

// Thing represents a Mainflux thing. Each thing is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Thing struct {
	ID        string
	GroupID   string
	ProfileID string
	Name      string
	Key       string
	Metadata  Metadata
}

// ThingsPage contains page related metadata as well as list of things that
// belong to this page.
type ThingsPage struct {
	PageMetadata
	Things []Thing
}

// ThingRepository specifies a thing persistence API.
type ThingRepository interface {
	// Save persists multiple things. Things are saved using a transaction. If one thing
	// fails then none will be saved. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, ths ...Thing) ([]Thing, error)

	// Update performs an update to the existing thing. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, t Thing) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(ctx context.Context, id, key string) error

	// RetrieveByID retrieves the thing having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(ctx context.Context, id string) (Thing, error)

	// RetrieveByKey returns thing ID for given thing key.
	RetrieveByKey(ctx context.Context, key string) (string, error)

	// RetrieveByGroupIDs retrieves the subset of things specified by given group ids.
	RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm PageMetadata) (ThingsPage, error)

	// RetrieveByProfile retrieves the subset of things assigned to the specified profile.
	RetrieveByProfile(ctx context.Context, prID string, pm PageMetadata) (ThingsPage, error)

	// Remove removes the things having the provided identifiers, that is owned
	// by the specified user.
	Remove(ctx context.Context, ids ...string) error

	// RetrieveAll retrieves all things for all users.
	RetrieveAll(ctx context.Context) ([]Thing, error)

	// RetrieveByAdmin retrieves all things for all users with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (ThingsPage, error)
}

// ThingCache contains thing caching interface.
type ThingCache interface {
	// Save stores pair thing key, thing id.
	Save(context.Context, string, string) error

	// ID returns thing ID for given key.
	ID(context.Context, string) (string, error)

	// Remove removes thing from cache.
	Remove(context.Context, string) error

	// SaveGroup stores group ID by given thing ID.
	SaveGroup(context.Context, string, string) error

	// ViewGroup returns group ID by given thing ID.
	ViewGroup(context.Context, string) (string, error)

	// RemoveGroup removes group ID by given thing ID.
	RemoveGroup(context.Context, string) error
}
