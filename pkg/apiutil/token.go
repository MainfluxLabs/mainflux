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

// ThingKey represents a Thing authentication key and its type
type ThingKey struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

func (tk ThingKey) Validate() error {
	if tk.Type != ThingKeyTypeExternal && tk.Type != ThingKeyTypeInternal {
		return ErrInvalidThingKeyType
	}

	if tk.Key == "" {
		return ErrBearerKey
	}

	return nil
}

// ExtractBearerToken returns value of the bearer token. If there is no bearer token - an empty value is returned.
func ExtractBearerToken(r *http.Request) string {
	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, BearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(token, BearerPrefix)
}

// ExtractThingKey returns the supplied thing key and its type, from the request's HTTP 'Authorization' header. If the provided key type is invalid
// an empty instance of ThingKey is returned.
func ExtractThingKey(r *http.Request) ThingKey {
	header := r.Header.Get("Authorization")

	switch {
	case strings.HasPrefix(header, ThingKeyPrefixInternal):
		return ThingKey{
			Type: ThingKeyTypeInternal,
			Key:  strings.TrimPrefix(header, ThingKeyPrefixInternal),
		}
	case strings.HasPrefix(header, ThingKeyPrefixExternal):
		return ThingKey{
			Type: ThingKeyTypeExternal,
			Key:  strings.TrimPrefix(header, ThingKeyPrefixExternal),
		}
	}

	return ThingKey{}
}
