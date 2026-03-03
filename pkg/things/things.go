// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	KeyTypeInternal = "internal"
	KeyTypeExternal = "external"
)

// ThingKey represents a Thing authentication key and its type.
type ThingKey struct {
	Value string `json:"key"`
	Type  string `json:"type"`
}

func (tk ThingKey) Validate() error {
	if tk.Type != KeyTypeExternal && tk.Type != KeyTypeInternal {
		return apiutil.ErrInvalidThingKeyType
	}

	if tk.Value == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

// ExtractThingKey returns the supplied thing key and its type, from the request's HTTP 'Authorization' header. If the provided key type is invalid
// an empty instance of ThingKey is returned.
func ExtractThingKey(r *http.Request) ThingKey {
	header := r.Header.Get("Authorization")

	switch {
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixInternal):
		return ThingKey{
			Type:  KeyTypeInternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixInternal),
		}
	case strings.HasPrefix(header, apiutil.ThingKeyPrefixExternal):
		return ThingKey{
			Type:  KeyTypeExternal,
			Value: strings.TrimPrefix(header, apiutil.ThingKeyPrefixExternal),
		}
	}

	return ThingKey{}
}
