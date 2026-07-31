// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package dbutil_test

import (
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/stretchr/testify/assert"
)

func TestGetOrderQuery(t *testing.T) {
	fields := map[string]string{
		"id":         "id",
		"name":       "LOWER(name)",
		"created_at": "created_at",
	}

	cases := []struct {
		desc     string
		order    string
		expected string
	}{
		{
			desc:     "known field id",
			order:    "id",
			expected: "id",
		},
		{
			desc:     "known field name with LOWER expression",
			order:    "name",
			expected: "LOWER(name)",
		},
		{
			desc:     "known field created_at",
			order:    "created_at",
			expected: "created_at",
		},
		{
			desc:     "empty order falls back to id",
			order:    "",
			expected: "id",
		},
		{
			desc:     "unknown order field falls back to id",
			order:    "unknown_field",
			expected: "id",
		},
		{
			desc:     "SQL injection attempt falls back to id",
			order:    "id; DROP TABLE users;--",
			expected: "id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := dbutil.GetOrderQuery(tc.order, fields)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestGetDirQuery(t *testing.T) {
	cases := []struct {
		desc     string
		dir      string
		expected string
	}{
		{
			desc:     "ascending direction",
			dir:      "asc",
			expected: "ASC",
		},
		{
			desc:     "descending direction",
			dir:      "desc",
			expected: "DESC",
		},
		{
			desc:     "empty direction defaults to DESC",
			dir:      "",
			expected: "DESC",
		},
		{
			desc:     "unknown direction defaults to DESC",
			dir:      "invalid",
			expected: "DESC",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := dbutil.GetDirQuery(tc.dir)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestGetNameQuery(t *testing.T) {
	cases := []struct {
		desc         string
		name         string
		expectedQ    string
		expectedName string
	}{
		{
			desc:         "non-empty name",
			name:         "test",
			expectedQ:    "LOWER(name) LIKE :name",
			expectedName: "%test%",
		},
		{
			desc:         "name with uppercase",
			name:         "Test",
			expectedQ:    "LOWER(name) LIKE :name",
			expectedName: "%test%",
		},
		{
			desc:         "empty name returns empty strings",
			name:         "",
			expectedQ:    "",
			expectedName: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			q, name := dbutil.GetNameQuery(tc.name)
			assert.Equal(t, tc.expectedQ, q)
			assert.Equal(t, tc.expectedName, name)
		})
	}
}

func TestGetOffsetLimitQuery(t *testing.T) {
	cases := []struct {
		desc     string
		limit    uint64
		expected string
	}{
		{
			desc:     "non-zero limit",
			limit:    10,
			expected: "LIMIT :limit OFFSET :offset",
		},
		{
			desc:     "zero limit returns empty string",
			limit:    0,
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := dbutil.GetOffsetLimitQuery(tc.limit)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestGetGroupIDsQuery(t *testing.T) {
	cases := []struct {
		desc     string
		ids      []string
		expected string
	}{
		{
			desc:     "single ID",
			ids:      []string{"id1"},
			expected: "group_id IN ('id1') ",
		},
		{
			desc:     "multiple IDs",
			ids:      []string{"id1", "id2", "id3"},
			expected: "group_id IN ('id1','id2','id3') ",
		},
		{
			desc:     "empty slice returns empty string",
			ids:      []string{},
			expected: "",
		},
		{
			desc:     "nil slice returns empty string",
			ids:      nil,
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := dbutil.GetGroupIDsQuery(tc.ids)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func TestBuildWhereClause(t *testing.T) {
	cases := []struct {
		desc     string
		filters  []string
		expected string
	}{
		{
			desc:     "single filter",
			filters:  []string{"name = 'test'"},
			expected: " WHERE name = 'test'",
		},
		{
			desc:     "multiple filters joined with AND",
			filters:  []string{"name = 'test'", "status = 'active'"},
			expected: " WHERE name = 'test' AND status = 'active'",
		},
		{
			desc:     "empty strings are excluded",
			filters:  []string{"", "name = 'test'", ""},
			expected: " WHERE name = 'test'",
		},
		{
			desc:     "all empty filters returns empty string",
			filters:  []string{"", ""},
			expected: "",
		},
		{
			desc:     "no filters returns empty string",
			filters:  []string{},
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := dbutil.BuildWhereClause(tc.filters...)
			assert.Equal(t, tc.expected, res)
		})
	}
}
