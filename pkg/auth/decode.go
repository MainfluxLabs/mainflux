// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

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

// Used as a key under which a domain.Identity type is stored in a context.Context.
type ctxKey struct{}

func WithIdentity(ctx context.Context, identity domain.Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, identity)
}

// IdentityFromCtx returns a domain.Identity associated with the passed context.
func IdentityFromCtx(ctx context.Context) (domain.Identity, bool) {
	identity, ok := ctx.Value(ctxKey{}).(domain.Identity)

	return identity, ok
}
