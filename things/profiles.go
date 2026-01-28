// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

// Profile represents a Mainflux "communication group". This group contains the
// things that can exchange messages between each other.
type Profile struct {
	ID       string
	GroupID  string
	Name     string
	Config   map[string]any
	Metadata map[string]any
}

type Config struct {
	ContentType string      `json:"content_type"`
	Transformer Transformer `json:"transformer"`
}

type Transformer struct {
	DataFilters  []string `json:"data_filters"`
	DataField    string   `json:"data_field"`
	TimeField    string   `json:"time_field"`
	TimeFormat   string   `json:"time_format"`
	TimeLocation string   `json:"time_location"`
}

// ProfilesPage contains page related metadata as well as list of profiles that
// belong to this page.
type ProfilesPage struct {
	Total    uint64
	Profiles []Profile
}

// ProfileRepository specifies a profile persistence API.
type ProfileRepository interface {
	// Save persists multiple profiles. Profiles are saved using a transaction. If one profile
	// fails then none will be saved. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, prs ...Profile) ([]Profile, error)

	// Update performs an update to the existing profile. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, c Profile) error

	// RetrieveByID retrieves the profile having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(ctx context.Context, id string) (Profile, error)

	// RetrieveByThing retrieves the profile connected to the given thing id.
	RetrieveByThing(ctx context.Context, thID string) (Profile, error)

	// Remove removes the profiles having the provided identifiers, that is owned
	// by the specified user.
	Remove(ctx context.Context, id ...string) error

	// BackupAll retrieves all profiles for all users.
	BackupAll(ctx context.Context) ([]Profile, error)

	// RetrieveAll retrieves all profiles for all users with pagination.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (ProfilesPage, error)

	// RetrieveByGroups retrieves the subset of profiles specified by given group ids.
	RetrieveByGroups(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (ProfilesPage, error)
}

// ProfileCache contains profile caching interface.
type ProfileCache interface {
	// SaveGroup stores group ID by given profile ID.
	SaveGroup(context.Context, string, string) error

	// RemoveGroup removes group ID by given profile ID.
	RemoveGroup(context.Context, string) error

	// ViewGroup returns group ID by given profile ID.
	ViewGroup(context.Context, string) (string, error)
}
