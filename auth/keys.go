// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

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
		return svc.tmpKey(svc.loginDuration, key)
	}
}

// loginKey opens a new session. Every login gets its own family, so revoking
// one session -- whether by logout or by reuse detection -- never disturbs the
// same user's other browsers or devices.
func (svc service) loginKey(ctx context.Context, key Key) (Key, string, error) {
	familyID, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return svc.issueSessionKey(ctx, key, familyID, key.IssuedAt)
}

// issueSessionKey mints a token and records it as the live member of its
// family. It is shared by the initial login and by every later rotation, which
// differ only in whether the family and session start are new or carried over.
func (svc service) issueSessionKey(ctx context.Context, key Key, familyID string, sessionStartAt time.Time) (Key, string, error) {
	jti, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}
	key.ID = jti

	session := Session{
		JTI:            jti,
		UserID:         key.IssuerID,
		FamilyID:       familyID,
		SessionStartAt: sessionStartAt,
	}
	if err := svc.sessions.Save(ctx, session); err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return svc.tmpKey(svc.loginDuration, key)
}

// Refresh rotates a login token. The presented token is revoked and replaced
// by a successor in the same session, so the session survives while the token
// string does not.
//
// Presenting a token that was already rotated means two parties hold the same
// token, which is treated as theft: the entire session is revoked, and the
// legitimate user has to log in again. Because each login gets its own family,
// that kill is scoped to the compromised session only.
func (svc service) Refresh(ctx context.Context, token string) (string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", errors.Wrap(errRefresh, err)
	}
	if key.Type != LoginKey || key.IssuerID == "" || key.ID == "" {
		return "", errors.Wrap(errRefresh, errors.ErrAuthentication)
	}

	session, err := svc.sessions.Retrieve(ctx, key.ID)
	if err != nil {
		return "", errors.Wrap(errRefresh, errors.ErrAuthentication)
	}

	if session.Revoked() {
		// A superseded token came back. Whoever holds it should not, so the
		// whole lineage descending from that login is killed.
		if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
			return "", errors.Wrap(errRefresh, err)
		}
		return "", errors.Wrap(errRefresh, ErrSessionReuse)
	}

	if getTimestamp().After(session.SessionStartAt.Add(svc.maxSessionAge)) {
		if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
			return "", errors.Wrap(errRefresh, err)
		}
		return "", errors.Wrap(errRefresh, ErrSessionExpired)
	}

	svc.purgeDeadSessions(ctx)

	next := Key{
		Type:     LoginKey,
		IssuerID: key.IssuerID,
		Subject:  key.Subject,
		IssuedAt: getTimestamp(),
	}

	// The successor is written before the predecessor is revoked: if the
	// revoke fails the worst case is a second live token in the family, which
	// a later rotation collapses, whereas the reverse order could leave the
	// session with no usable token at all.
	_, secret, err := svc.issueSessionKey(ctx, next, session.FamilyID, session.SessionStartAt)
	if err != nil {
		return "", errors.Wrap(errRefresh, err)
	}

	if err := svc.sessions.Revoke(ctx, session.JTI, getTimestamp()); err != nil {
		return "", errors.Wrap(errRefresh, err)
	}

	return secret, nil
}

// Logout ends the session the token belongs to, leaving the user's other
// sessions alone.
func (svc service) Logout(ctx context.Context, token string) error {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return errors.Wrap(errLogout, err)
	}
	if key.Type != LoginKey || key.ID == "" {
		return errors.Wrap(errLogout, errors.ErrAuthentication)
	}

	session, err := svc.sessions.Retrieve(ctx, key.ID)
	if err != nil {
		return errors.Wrap(errLogout, errors.ErrAuthentication)
	}

	if err := svc.sessions.RevokeFamily(ctx, session.FamilyID, getTimestamp()); err != nil {
		return errors.Wrap(errLogout, err)
	}

	return nil
}

// LogoutAll ends every session belonging to the owner of the token.
func (svc service) LogoutAll(ctx context.Context, token string) error {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return errors.Wrap(errLogout, err)
	}

	if err := svc.sessions.RevokeByUser(ctx, issuerID, getTimestamp()); err != nil {
		return errors.Wrap(errLogout, err)
	}

	return nil
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

	//nolint:errcheck // best-effort cleanup
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
