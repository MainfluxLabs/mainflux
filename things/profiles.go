// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
)

// Profile is an alias for the shared domain type.
type Profile = domainthings.Profile

// ProfilesPage is an alias for the shared domain type.
type ProfilesPage = domainthings.ProfilesPage

// Config and Transformer are aliases for the shared domain types.
type Config = domainthings.Config
type Transformer = domainthings.Transformer

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
