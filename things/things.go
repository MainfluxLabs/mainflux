// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

// Metadata to be used for Mainflux thing or profile for customized
// describing of particular thing or profile.
type Metadata map[string]any

// Thing represents a Mainflux thing. Each thing is owned by one user, and
// it is assigned with the unique identifier and (temporary) access key.
type Thing struct {
	ID          string
	GroupID     string
	ProfileID   string
	Name        string
	Key         string
	ExternalKey string
	Metadata    Metadata
}

// ThingsPage contains page related metadata as well as list of things that
// belong to this page.
type ThingsPage struct {
	Total  uint64
	Things []Thing
}

const (
	KeyTypeInternal = "internal"
	KeyTypeExternal = "external"
)

// ThingKey represents a Thing authentication key and its type
type ThingKey struct {
	Value string `json:"key"`
	Type  string `json:"type"`
}

func (tk ThingKey) Validate() error {
	if tk.Type != KeyTypeExternal && tk.Type != KeyTypeInternal {
		return apiutil.ErrInvalidThingKeyType
	}

	if tk.Value == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

// ExtractThingKey returns the supplied thing key and its type, from the request's HTTP 'Authorization' header. If the provided key type is invalid
// an empty instance of ThingKey is returned.
func ExtractThingKey(r *http.Request) ThingKey {
	header := r.Header.Get("Authorization")

	switch {
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixInternal):
		return ThingKey{
			Type:  KeyTypeInternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixInternal),
		}
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixExternal):
		return ThingKey{
			Type:  KeyTypeExternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixExternal),
		}
	}

	return ThingKey{}
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

	// UpdateGroupAndProfile performs a an update of an existing Thing's Profile and/or Group. A non-nil error is
	// returned to indicate operation failure.
	UpdateGroupAndProfile(ctx context.Context, t Thing) error

	// RetrieveByID retrieves the thing having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(ctx context.Context, id string) (Thing, error)

	// RetrieveByKey returns thing ID for given thing key based on its type.
	RetrieveByKey(ctx context.Context, key ThingKey) (string, error)

	// RetrieveByGroups retrieves the subset of things specified by given group ids.
	RetrieveByGroups(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (ThingsPage, error)

	// RetrieveByProfile retrieves the subset of things assigned to the specified profile.
	RetrieveByProfile(ctx context.Context, prID string, pm apiutil.PageMetadata) (ThingsPage, error)

	// Remove removes the things having the provided identifiers, that is owned
	// by the specified user.
	Remove(ctx context.Context, ids ...string) error

	// BackupAll retrieves all things for all users.
	BackupAll(ctx context.Context) ([]Thing, error)

	// RetrieveAll retrieves all things for all users with pagination.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (ThingsPage, error)

	// UpdateExternalKey sets/updates the external key of the Thing identified by `thingID`.
	UpdateExternalKey(ctx context.Context, key, thingID string) error

	// RemoveExternalKey removes an external key from the thing identified by `thingID`.
	RemoveExternalKey(ctx context.Context, thingID string) error
}

// ThingCache contains thing caching interface.
type ThingCache interface {
	// Save stores the pair (thing key, thing id).
	Save(ctx context.Context, key ThingKey, thingID string) error

	// ID returns thing ID for a given thing key.
	ID(ctx context.Context, key ThingKey) (string, error)

	// RemoveThing removes thing from cache.
	RemoveThing(context.Context, string) error

	// RemoveKey removes a specific thing key from the cache.
	RemoveKey(ctx context.Context, key ThingKey) error

	// SaveGroup stores group ID by given thing ID.
	SaveGroup(context.Context, string, string) error

	// ViewGroup returns group ID by given thing ID.
	ViewGroup(context.Context, string) (string, error)

	// RemoveGroup removes group ID by given thing ID.
	RemoveGroup(context.Context, string) error
}
