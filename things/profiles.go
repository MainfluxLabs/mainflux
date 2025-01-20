// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
)

// Profile represents a Mainflux "communication group". This group contains the
// things that can exchange messages between each other.
type Profile struct {
	ID       string
	GroupID  string
	Name     string
	Config   map[string]interface{}
	Metadata map[string]interface{}
}

type Config struct {
	ContentType string      `json:"content_type"`
	Write       bool        `json:"write"`
	WebhookID   string      `json:"webhook_id"`
	Transformer Transformer `json:"transformer"`
	SmtpID      string      `json:"smtp_id"`
	SmppID      string      `json:"smpp_id"`
}

type Transformer struct {
	DataFilters  []string `json:"data_filters"`
	DataField    string   `json:"data_field"`
	TimeField    string   `json:"time_field"`
	TimeFormat   string   `json:"time_format"`
	TimeLocation string   `json:"time_location"`
}

type Notifier struct {
	ID       string
	GroupID  string
	Name     string
	Contacts []string
	Metadata Metadata
}

type NotifiersPage struct {
	PageMetadata
	Notifiers []Notifier
}

// ProfilesPage contains page related metadata as well as list of profiles that
// belong to this page.
type ProfilesPage struct {
	PageMetadata
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

	// RetrieveAll retrieves all profiles for all users.
	RetrieveAll(ctx context.Context) ([]Profile, error)

	// RetrieveByAdmin retrieves all profiles for all users with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (ProfilesPage, error)

	// RetrieveByGroupIDs retrieves the subset of profiles specified by given group ids.
	RetrieveByGroupIDs(ctx context.Context, groupIDs []string, pm PageMetadata) (ProfilesPage, error)
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
