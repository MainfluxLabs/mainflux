// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// ErrShadowNotFound indicates that a shadow does not exist for the thing.
var ErrShadowNotFound = errors.New("shadow not found")

// State is a free-form set of key/value pairs describing device state.
// A nil value for a key signals that the key should be deleted.
type State map[string]any

// Shadow is the persisted state document for a single thing. Exactly one
// shadow exists per thing. Delta is not stored; it is derived from Desired
// and Reported via computeDelta and populated by the service on read.
type Shadow struct {
	ThingID   string
	Desired   State
	Reported  State
	Delta     State
	Version   uint64
	Timestamp int64
}

// ShadowRepository specifies the persistence API for shadows.
type ShadowRepository interface {
	// Upsert creates or replaces the shadow for a thing and returns the
	// stored document (with the incremented version).
	Upsert(ctx context.Context, shadow Shadow) (Shadow, error)

	// RetrieveByThing returns the shadow for the given thing ID, or
	// ErrShadowNotFound if none exists.
	RetrieveByThing(ctx context.Context, thingID string) (Shadow, error)

	// Remove deletes the shadow for the given thing ID.
	Remove(ctx context.Context, thingID string) error

	// RetrieveAll returns every stored shadow.
	RetrieveAll(ctx context.Context) ([]Shadow, error)
}
