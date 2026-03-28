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

// ExtractThingKey returns the thing key and its type from the request's HTTP 'Authorization' header.
// If the provided key type is invalid, an empty ThingKey is returned.
func ExtractThingKey(r *http.Request) domain.ThingKey {
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
