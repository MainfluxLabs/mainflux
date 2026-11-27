// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/MainfluxLabs/mainflux/rules/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRunsPageMetadata(t *testing.T) {
	cases := []struct {
		desc  string
		query string
		order string
	}{
		{
			desc:  "build rule runs page metadata without order",
			query: "",
			order: "started_at",
		},
		{
			desc:  "build rule runs page metadata with order",
			query: "?order=id",
			order: "id",
		},
	}

	for _, tc := range cases {
		r := httptest.NewRequest("GET", "/rules/id/runs"+tc.query, nil)
		pm, err := api.BuildRunsPageMetadata(r)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.order, pm.Order, fmt.Sprintf("%s: expected order %s got %s", tc.desc, tc.order, pm.Order))
	}
}
