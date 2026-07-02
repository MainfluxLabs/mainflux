// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows

import (
	"context"
	"maps"
)

// State is a free-form set of key/value pairs describing device state.
type State map[string]any

// Shadow is the persisted state of a single thing.
// Delta is derived from Desired and Reported and populated
// by the service on read.
type Shadow struct {
	ThingID    string
	Desired    State
	Reported   State
	Delta      State
	ReportedAt int64
	UpdatedAt  int64
}

// ShadowRepository specifies the persistence API for shadows.
type ShadowRepository interface {
	// Upsert creates or replaces the shadow for a thing and returns the
	// stored shadow.
	Upsert(ctx context.Context, shadow Shadow) (Shadow, error)

	// RetrieveByThing returns the shadow for the given thing ID.
	RetrieveByThing(ctx context.Context, thingID string) (Shadow, error)

	// Remove deletes the shadow for the given thing ID.
	Remove(ctx context.Context, thingID string) error
}

// equalState reports whether two JSON-shaped shadow values are deeply equal.
func equalState(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			w, ok := bv[k]
			if !ok || !equalState(v, w) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equalState(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// computeDelta returns the subset of desired that the reported state has not
// yet matched: keys present in desired whose value differs from, or is absent
// in, reported. Keys present only in reported are not part of the delta. The
// result is nil when desired and reported already agree on every desired key.
func computeDelta(desired, reported State) State {
	var delta State
	for k, dv := range desired {
		rv, ok := reported[k]
		if ok && equalState(dv, rv) {
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
		if cur, ok := merged[k]; !ok || !equalState(cur, v) {
			merged[k] = v
			changed = true
		}
	}

	return merged, changed
}
