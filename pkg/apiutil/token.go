// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
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

// ExtractThingKeyFromHTTPHeader returns the thing key and its type from the request's HTTP 'Authorization' header.
// If the provided key type is invalid, an empty ThingKey is returned.
func ExtractThingKeyFromHTTPHeader(r *http.Request) domain.ThingKey {
	header := r.Header.Get("Authorization")

	switch {
	case strings.HasPrefix(header, ThingKeyPrefixInternal):
		return domain.ThingKey{
			Type:  domain.KeyTypeInternal,
			Value: strings.TrimPrefix(header, ThingKeyPrefixInternal),
		}
	case strings.HasPrefix(header, ThingKeyPrefixExternal):
		return domain.ThingKey{
			Type:  domain.KeyTypeExternal,
			Value: strings.TrimPrefix(header, ThingKeyPrefixExternal),
		}
	}

	return domain.ThingKey{}
}

// ValidateInviteeRole returns an error if the role is not a valid invitee role
// (OrgAdmin, OrgEditor, or OrgViewer). OrgOwner cannot be assigned via invite.
func ValidateInviteeRole(role string) error {
	if role != domain.OrgAdmin && role != domain.OrgEditor && role != domain.OrgViewer {
		return ErrInvalidRole
	}
	return nil
}

// ValidateThingKey returns an API validation error if the thing key is invalid.
func ValidateThingKey(key domain.ThingKey) error {
	if key.Type != domain.KeyTypeExternal && key.Type != domain.KeyTypeInternal {
		return ErrInvalidThingKeyType
	}
	if key.Value == "" {
		return ErrBearerKey
	}
	return nil
}
