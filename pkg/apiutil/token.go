// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"net/http"
	"strings"
)

const (
	// BearerPrefix represents the token prefix for Bearer authentication scheme.
	BearerPrefix = "Bearer "
	// ThingKeyPrefixInternal represents the key prefix for Thing authentication based on an internal key.
	ThingKeyPrefixInternal = "Thing "
	// ThingKeyPrefixExternal represents the key prefix for Thing authentication based on an externaly defined key.
	ThingKeyPrefixExternal = "External "
)

// ExtractBearerToken returns value of the bearer token. If there is no bearer token - an empty value is returned.
func ExtractBearerToken(r *http.Request) string {
	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, BearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(token, BearerPrefix)
}
