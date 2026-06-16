// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows

import (
	"maps"
	"reflect"
)

// ComputeDelta returns the subset of desired that the reported state has not
// yet matched: keys present in desired whose value differs from, or is absent
// in, reported. Keys present only in reported are not part of the delta. The
// result is nil when desired and reported already agree on every desired key.
func ComputeDelta(desired, reported State) State {
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

// MergeState applies patch onto base and reports whether anything changed.
// A nil value in patch deletes the corresponding key. The merge is shallow:
// a non-nil value replaces the existing value for that key outright. base is
// not mutated; the merged copy is returned.
func MergeState(base, patch State) (merged State, changed bool) {
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
