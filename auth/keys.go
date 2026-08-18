// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	// ErrInvalidKeyIssuedAt indicates that the Key is being used before it's issued.
	ErrInvalidKeyIssuedAt = errors.New("invalid issue time")

	// ErrKeyExpired indicates that the Key is expired.
	ErrKeyExpired = errors.New("use of expired key")

	// ErrAPIKeyExpired indicates that the Key is expired
	// and that the key type is API key.
	ErrAPIKeyExpired = errors.New("use of expired API key")
)

// KeyOrderFields maps API-facing order keys to SQL column expressions for the keys table.
var KeyOrderFields = map[string]string{
	"id":        "id",
	"issued_at": "issued_at",
}

// Domain type aliases
type (
	Key       = domain.Key
	Identity  = domain.Identity
	KeysPage  = domain.KeysPage
	TokenPair = domain.TokenPair
)

// Key type constants (aliases for domain).
const (
	LoginKey    = domain.LoginKey
	RecoveryKey = domain.RecoveryKey
	APIKey      = domain.APIKey
	RefreshKey  = domain.RefreshKey
)

// Keys specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Keys interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (Key, string, error)

	// Refresh exchanges a valid refresh key for a freshly minted login key,
	// letting clients extend a session without re-sending credentials.
	Refresh(ctx context.Context, refreshToken string) (Key, string, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(ctx context.Context, token, id string) error

	// RetrieveKey retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	RetrieveKey(ctx context.Context, token, id string) (Key, error)

	// ListAPIKeys retrieves API keys.
	ListAPIKeys(ctx context.Context, token string, pm PageMetadata) (KeysPage, error)
}

// KeyRepository specifies Key persistence API.
type KeyRepository interface {
	// Save persists the Key. A non-nil error is returned to indicate
	// operation failure
	Save(context.Context, Key) (string, error)

	// Retrieve retrieves Key by its unique identifier.
	Retrieve(context.Context, string, string) (Key, error)

	// Remove removes Key with provided ID.
	Remove(context.Context, string, string) error

	// RetrieveAPIKeys retrieves all API Keys with pagination.
	RetrieveAPIKeys(ctx context.Context, issuerID string, pm PageMetadata) (KeysPage, error)
}

func (svc service) Issue(ctx context.Context, token string, key Key) (Key, string, error) {
	if key.IssuedAt.IsZero() {
		return Key{}, "", ErrInvalidKeyIssuedAt
	}
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, token, key)
	case RecoveryKey:
		return svc.tmpKey(recoveryDuration, key)
	case RefreshKey:
		return svc.tmpKey(svc.refreshDuration, key)
	default:
		return svc.tmpKey(svc.loginDuration, key)
	}
}

// Refresh mints a new login key from a valid refresh key. The refresh key
// itself is deliberately not re-issued: since refresh keys are stateless and
// therefore not revocable, sliding the expiry on every call would let a leaked
// key renew itself indefinitely. A session is capped at refreshDuration from
// the original login.
func (svc service) Refresh(ctx context.Context, refreshToken string) (Key, string, error) {
	key, err := svc.tokenizer.Parse(refreshToken)
	if err != nil {
		return Key{}, "", errors.Wrap(errRefresh, err)
	}

	// Only a refresh key may be exchanged; an access token must not be able to
	// extend its own lifetime.
	if key.Type != RefreshKey || key.IssuerID == "" {
		return Key{}, "", errors.Wrap(errRefresh, errors.ErrAuthentication)
	}

	access := Key{
		Type:     LoginKey,
		IssuerID: key.IssuerID,
		Subject:  key.Subject,
		IssuedAt: getTimestamp(),
	}

	return svc.tmpKey(svc.loginDuration, access)
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) RetrieveKey(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.Retrieve(ctx, issuerID, id)
}

func (svc service) ListAPIKeys(ctx context.Context, token string, pm PageMetadata) (KeysPage, error) {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return KeysPage{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.RetrieveAPIKeys(ctx, issuerID, pm)
}

func (svc service) userKey(ctx context.Context, token string, key Key) (Key, string, error) {
	id, sub, err := svc.login(token)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	key.IssuerID = id
	if key.Subject == "" {
		key.Subject = sub
	}

	keyID, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}
	key.ID = keyID

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return key, secret, nil
}
