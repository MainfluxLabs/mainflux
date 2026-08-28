// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// purgeInterval throttles the opportunistic cleanup of dead session rows, so
// that at most one refresh per interval pays for it.
const purgeInterval = time.Hour

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
	Key      = domain.Key
	Identity = domain.Identity
	KeysPage = domain.KeysPage
)

// Key type constants (aliases for domain).
const (
	LoginKey    = domain.LoginKey
	RecoveryKey = domain.RecoveryKey
	APIKey      = domain.APIKey
)

// Keys specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Keys interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (Key, string, error)

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
	case LoginKey:
		return svc.loginKey(ctx, key)
	default:
		return Key{}, "", apiutil.ErrInvalidAuthKey
	}
}

func (svc service) loginKey(ctx context.Context, key Key) (Key, string, error) {
	if key.IssuerID == "" {
		return Key{}, "", errors.Wrap(errIssueUser, apiutil.ErrInvalidAPIKey)
	}

	familyID, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return svc.issueSessionKey(ctx, key, familyID, key.IssuedAt)
}

// issueSessionKey mints a token and records it as the live member of its family.
func (svc service) issueSessionKey(ctx context.Context, key Key, familyID string, startedAt time.Time) (Key, string, error) {
	jti, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}
	key.ID = jti

	session := Session{
		JTI:       jti,
		UserID:    key.IssuerID,
		FamilyID:  familyID,
		StartedAt: startedAt,
	}
	if err := svc.sessions.Save(ctx, session); err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return svc.tmpKey(svc.loginDuration, key)
}

// Refresh rotates a login token, replacing it with a successor in the same
// session. Presenting a token that was already rotated means two parties hold
// it, which is treated as theft: the whole session is revoked. Because each
// login gets its own family, that kill is scoped to the compromised session.
func (svc service) Refresh(ctx context.Context, token string) (string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", errors.Wrap(errRefresh, err)
	}

	if key.Type != LoginKey || key.IssuerID == "" || key.ID == "" {
		return "", errors.Wrap(errRefresh, errors.ErrAuthentication)
	}

	jti, err := svc.idProvider.ID()
	if err != nil {
		return "", errors.Wrap(errRefresh, err)
	}

	_, secret, err := svc.tmpKey(svc.loginDuration, Key{
		ID:       jti,
		Type:     LoginKey,
		IssuerID: key.IssuerID,
		Subject:  key.Subject,
		IssuedAt: getTimestamp(),
	})

	if err != nil {
		return "", errors.Wrap(errRefresh, err)
	}

	session, won, err := svc.sessions.Rotate(ctx, key.ID, jti, getTimestamp())
	if err != nil {
		if errors.Contains(err, dbutil.ErrNotFound) {
			return "", errors.Wrap(errRefresh, errors.ErrAuthentication)
		}

		return "", errors.Wrap(errRefresh, err)
	}

	if !won {
		if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
			return "", errors.Wrap(errRefresh, err)
		}

		return "", errors.Wrap(errRefresh, ErrSessionReuse)
	}

	if getTimestamp().After(session.StartedAt.Add(svc.maxSessionAge)) {
		if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
			return "", errors.Wrap(errRefresh, err)
		}

		return "", errors.Wrap(errRefresh, ErrSessionExpired)
	}

	svc.purgeDeadSessions(ctx)

	return secret, nil
}

func (svc service) Logout(ctx context.Context, token string) error {
	session, err := svc.liveSession(ctx, token)
	if err != nil {
		return errors.Wrap(errLogout, err)
	}

	if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
		return errors.Wrap(errLogout, err)
	}

	return nil
}

func (svc service) LogoutAll(ctx context.Context, token string) error {
	session, err := svc.liveSession(ctx, token)
	if err != nil {
		return errors.Wrap(errLogout, err)
	}

	if err := svc.sessions.RevokeByUser(ctx, session.UserID, getTimestamp()); err != nil {
		return errors.Wrap(errLogout, err)
	}

	return nil
}

func (svc service) RevokeUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.Wrap(errLogout, apiutil.ErrMissingUserID)
	}

	if err := svc.sessions.RevokeByUser(ctx, userID, getTimestamp()); err != nil {
		return errors.Wrap(errLogout, err)
	}

	return nil
}

// liveSession resolves a login token to the session backing it, rejecting one
// already revoked. A signature only proves the token was minted at some point,
// so without this check a token that was logged out, or killed as stolen, could
// go on ending its owner's sessions until it expired.
func (svc service) liveSession(ctx context.Context, token string) (Session, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return Session{}, err
	}
	if key.Type != LoginKey || key.IssuerID == "" || key.ID == "" {
		return Session{}, errors.ErrAuthentication
	}

	session, err := svc.sessions.Retrieve(ctx, key.ID)
	if err != nil {
		return Session{}, errors.ErrAuthentication
	}
	if session.Revoked() {
		return Session{}, errors.ErrAuthentication
	}

	return session, nil
}

// purgeDeadSessions drops rows that can no longer belong to a live session.
// It rides along with refresh rather than a separate job, and is throttled so
// only one caller per purgeInterval pays the cost. Failures are ignored: this
// is housekeeping, not part of the refresh contract.
func (svc service) purgeDeadSessions(ctx context.Context) {
	now := getTimestamp()
	last := svc.lastPurge.Load()
	if now.Sub(time.Unix(0, last)) < purgeInterval {
		return
	}

	if !svc.lastPurge.CompareAndSwap(last, now.UnixNano()) {
		return
	}

	svc.sessions.RemoveExpired(ctx, now.Add(-svc.maxSessionAge))
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
