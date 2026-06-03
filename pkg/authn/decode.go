// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// EmailFromToken extracts the email (subject) from a JWT token without
// verifying the signature. Returns "" if the token is malformed.
func EmailFromToken(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Subject string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Subject
}

// Used as a key under which a domain.Identity type is stored in a Context.
type identityCtxKey struct{}

// Used as a key under which an auth token string is stored in a Context.
type tokenCtxKey struct{}

// IdentityFromCtx extracts a domain.Identity associated with the passed Context, if it exists.
func IdentityFromCtx(ctx context.Context) (domain.Identity, bool) {
	identity, ok := ctx.Value(identityCtxKey{}).(domain.Identity)

	return identity, ok
}
