// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows

import (
	"context"
	"maps"
	"reflect"
)

// State is a free-form set of key/value pairs describing device state.
// A nil value for a key signals that the key should be deleted.
type State map[string]any

// Shadow is the persisted state document for a single thing. Exactly one
// shadow exists per thing. Delta is not stored; it is derived from Desired
// and Reported via computeDelta and populated by the service on read.
type Shadow struct {
	ThingID  string
	Desired  State
	Reported State
	Delta    State
	ReportedAt int64
	UpdatedAt  int64
}

// ShadowRepository specifies the persistence API for shadows.
type ShadowRepository interface {
	// Upsert creates or replaces the shadow for a thing and returns the
	// stored document.
	Upsert(ctx context.Context, shadow Shadow) (Shadow, error)

	// RetrieveByThing returns the shadow for the given thing ID, or an empty
	// shadow if none exists yet.
	RetrieveByThing(ctx context.Context, thingID string) (Shadow, error)

	// Remove deletes the shadow for the given thing ID.
	Remove(ctx context.Context, thingID string) error
}

// computeDelta returns the subset of desired that the reported state has not
// yet matched: keys present in desired whose value differs from, or is absent
// in, reported. Keys present only in reported are not part of the delta. The
// result is nil when desired and reported already agree on every desired key.
func computeDelta(desired, reported State) State {
	var delta State
	for k, dv := range desired {
		rv, ok := reported[k]
		if ok && reflect.DeepEqual(dv, rv) {
			continue
		}
		if delta == nil {
			delta = State{}
		}
		delta[k] = dv
	}
	return delta
}

// mergeState applies patch onto base and reports whether anything changed.
// A nil value in patch deletes the corresponding key. The merge is shallow:
// a non-nil value replaces the existing value for that key outright. base is
// not mutated; the merged copy is returned.
func mergeState(base, patch State) (merged State, changed bool) {
	merged = State{}
	maps.Copy(merged, base)

	for k, v := range patch {
		if v == nil {
			if _, ok := merged[k]; ok {
				delete(merged, k)
				changed = true
			}
			continue
		}
		if cur, ok := merged[k]; !ok || !reflect.DeepEqual(cur, v) {
			merged[k] = v
			changed = true
		}
	}

	return merged, changed
}
