// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows_test

import (
	"testing"

	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/stretchr/testify/assert"
)

func TestComputeDelta(t *testing.T) {
	cases := []struct {
		desc     string
		desired  shadows.State
		reported shadows.State
		delta    shadows.State
	}{
		{
			desc:     "desired key absent from reported",
			desired:  shadows.State{"fan": "ON"},
			reported: shadows.State{"temperature": 22.0},
			delta:    shadows.State{"fan": "ON"},
		},
		{
			desc:     "desired key differs from reported",
			desired:  shadows.State{"fan": "ON"},
			reported: shadows.State{"fan": "OFF"},
			delta:    shadows.State{"fan": "ON"},
		},
		{
			desc:     "desired key matches reported",
			desired:  shadows.State{"fan": "ON"},
			reported: shadows.State{"fan": "ON"},
			delta:    nil,
		},
		{
			desc:     "key only in reported is ignored",
			desired:  shadows.State{},
			reported: shadows.State{"temperature": 22.0},
			delta:    nil,
		},
		{
			desc:     "mixed: one matching, one differing",
			desired:  shadows.State{"fan": "ON", "mode": "auto"},
			reported: shadows.State{"fan": "ON", "mode": "manual"},
			delta:    shadows.State{"mode": "auto"},
		},
		{
			desc:     "nested value differs",
			desired:  shadows.State{"loc": map[string]any{"lat": 1.0, "lon": 2.0}},
			reported: shadows.State{"loc": map[string]any{"lat": 1.0, "lon": 9.0}},
			delta:    shadows.State{"loc": map[string]any{"lat": 1.0, "lon": 2.0}},
		},
		{
			desc:     "nested value matches",
			desired:  shadows.State{"loc": map[string]any{"lat": 1.0, "lon": 2.0}},
			reported: shadows.State{"loc": map[string]any{"lat": 1.0, "lon": 2.0}},
			delta:    nil,
		},
	}

	for _, tc := range cases {
		delta := shadows.ComputeDelta(tc.desired, tc.reported)
		assert.Equal(t, tc.delta, delta, tc.desc)
	}
}

func TestMergeState(t *testing.T) {
	cases := []struct {
		desc    string
		base    shadows.State
		patch   shadows.State
		merged  shadows.State
		changed bool
	}{
		{
			desc:    "new key added",
			base:    shadows.State{"fan": "ON"},
			patch:   shadows.State{"temperature": 22.0},
			merged:  shadows.State{"fan": "ON", "temperature": 22.0},
			changed: true,
		},
		{
			desc:    "existing key updated",
			base:    shadows.State{"fan": "OFF"},
			patch:   shadows.State{"fan": "ON"},
			merged:  shadows.State{"fan": "ON"},
			changed: true,
		},
		{
			desc:    "same value is a no-op",
			base:    shadows.State{"fan": "ON"},
			patch:   shadows.State{"fan": "ON"},
			merged:  shadows.State{"fan": "ON"},
			changed: false,
		},
		{
			desc:    "nil value deletes existing key",
			base:    shadows.State{"fan": "ON", "mode": "auto"},
			patch:   shadows.State{"mode": nil},
			merged:  shadows.State{"fan": "ON"},
			changed: true,
		},
		{
			desc:    "nil value for absent key is a no-op",
			base:    shadows.State{"fan": "ON"},
			patch:   shadows.State{"mode": nil},
			merged:  shadows.State{"fan": "ON"},
			changed: false,
		},
		{
			desc:    "empty patch is a no-op",
			base:    shadows.State{"fan": "ON"},
			patch:   shadows.State{},
			merged:  shadows.State{"fan": "ON"},
			changed: false,
		},
	}

	for _, tc := range cases {
		merged, changed := shadows.MergeState(tc.base, tc.patch)
		assert.Equal(t, tc.merged, merged, tc.desc)
		assert.Equal(t, tc.changed, changed, tc.desc)
	}
}

func TestMergeStateDoesNotMutateBase(t *testing.T) {
	base := shadows.State{"fan": "OFF"}
	_, _ = shadows.MergeState(base, shadows.State{"fan": "ON", "mode": "auto"})
	assert.Equal(t, shadows.State{"fan": "OFF"}, base, "base must not be mutated")
}
