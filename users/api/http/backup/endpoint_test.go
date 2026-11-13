// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBackupResponse(t *testing.T) {
	admin := users.User{ID: "574106f7-030e-4881-8ab0-151195c29f94", Email: "admin@example.com"}

	res := buildBackupResponse(admin, nil, nil)

	data, err := json.Marshal(res)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	var body map[string]json.RawMessage
	err = json.Unmarshal(data, &body)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	for _, key := range []string{"users", "identities"} {
		msg := fmt.Sprintf("%s: expected [] got %s\n", key, body[key])
		assert.Equal(t, "[]", string(body[key]), msg)
	}
}
