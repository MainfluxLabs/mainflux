// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	recoveryDuration = 5 * time.Minute

	// Re-export role constants from domain for backward compatibility.
	Admin   = domain.OrgAdmin
	Owner   = domain.OrgOwner
	Editor  = domain.OrgEditor
	Viewer  = domain.OrgViewer
	RootSub = domain.RootSub
	OrgSub  = domain.OrgSub
)

// Domain type aliases
type (
	AuthzReq = domain.AuthzReq
)

var (
	// ErrRetrieveMembershipsByOrg indicates that retrieving memberships by org failed.
	ErrRetrieveMembershipsByOrg = errors.New("failed to retrieve memberships by org")

	// ErrRetrieveOrgsByMembership indicates that retrieving orgs by membership failed.
	ErrRetrieveOrgsByMembership = errors.New("failed to retrieve orgs by membership")

	errIssueUser = errors.New("failed to issue new login key")
	errIssueTmp  = errors.New("failed to issue new temporary key")
	errRevoke    = errors.New("failed to remove key")
	errRetrieve  = errors.New("failed to retrieve key data")
	errIdentify  = errors.New("failed to validate token")
	errRefresh   = errors.New("failed to refresh login key")
	errLogout    = errors.New("failed to end session")

	// ErrSessionReuse indicates that an already rotated token was presented
	// again, which is treated as theft: the whole session is killed.
	ErrSessionReuse = errors.New("refresh token reuse detected, session revoked")

	// ErrSessionExpired indicates the session outlived its maximum age and
	// cannot be extended any further.
	ErrSessionExpired = errors.New("session has reached its maximum lifetime")
	errUnknownSubject = errors.New("unknown subject")
)

type PageMetadata struct {
	Total    uint64         `json:"total,omitempty"`
	Offset   uint64         `json:"offset,omitempty"`
	Limit    uint64         `json:"limit,omitempty"`
	Order    string         `json:"order,omitempty"`
	Dir      string         `json:"dir,omitempty"`
	State    string         `json:"state,omitempty"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Email    string         `json:"email,omitempty"`
	Role     string         `json:"role,omitempty"`
}

// Authn specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Authn interface {
	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, token string) (Identity, error)
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	Authorize(ctx context.Context, ar AuthzReq) error
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Service interface {
	Authn
	Authz
	Roles
	Orgs
	OrgMemberships
	Invites
	Keys
	Sessions
}

var _ Service = (*service)(nil)

type service struct {
	orgs           OrgRepository
	users          domain.UsersClient
	things         domain.ThingsClient
	keys           KeyRepository
	sessions       SessionRepository
	roles          RolesRepository
	memberships    OrgMembershipsRepository
	invites        OrgInvitesRepository
	email          Emailer
	idProvider     uuid.IDProvider
	tokenizer      Tokenizer
	loginDuration  time.Duration
	maxSessionAge  time.Duration
	inviteDuration time.Duration
	lastPurge      *atomic.Int64
}

// New instantiates the auth service implementation.
func New(orgs OrgRepository, tc domain.ThingsClient, uc domain.UsersClient, keys KeyRepository, sessions SessionRepository, roles RolesRepository,
	memberships OrgMembershipsRepository, invites OrgInvitesRepository, emailer Emailer, idp uuid.IDProvider, tokenizer Tokenizer, loginDuration time.Duration, maxSessionAge time.Duration, inviteDuration time.Duration) Service {
	return &service{
		tokenizer:      tokenizer,
		things:         tc,
		orgs:           orgs,
		users:          uc,
		keys:           keys,
		sessions:       sessions,
		roles:          roles,
		memberships:    memberships,
		invites:        invites,
		email:          emailer,
		idProvider:     idp,
		loginDuration:  loginDuration,
		maxSessionAge:  maxSessionAge,
		inviteDuration: inviteDuration,
		lastPurge:      &atomic.Int64{},
	}
}

func (svc service) Authorize(ctx context.Context, ar AuthzReq) error {
	switch ar.Subject {
	case RootSub:
		return svc.isAdmin(ctx, ar.Token)
	case OrgSub:
		return svc.canAccessOrg(ctx, ar.Token, ar.Object, ar.Action)
	default:
		return errUnknownSubject
	}
}

func (svc service) Identify(ctx context.Context, token string) (Identity, error) {
	return svc.identify(ctx, token)
}

func (svc service) identify(ctx context.Context, token string) (Identity, error) {
	key, err := svc.tokenizer.Parse(token)
	if err == ErrAPIKeyExpired {
		err = svc.keys.Remove(ctx, key.IssuerID, key.ID)
		return Identity{}, errors.Wrap(ErrAPIKeyExpired, err)
	}
	if err != nil {
		return Identity{}, errors.Wrap(errIdentify, err)
	}

	switch key.Type {
	case RecoveryKey:
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	case LoginKey:
		// Login tokens are revocable, so a valid signature is not enough: the
		// session row has the last word. This runs on every authenticated
		// request, which is what makes logout and reuse kills take effect
		// immediately rather than at the token's natural expiry.
		if key.ID == "" {
			return Identity{}, errors.Wrap(errIdentify, errors.ErrAuthentication)
		}
		session, err := svc.sessions.Retrieve(ctx, key.ID)
		if err != nil {
			return Identity{}, errors.Wrap(errIdentify, errors.ErrAuthentication)
		}
		if session.Revoked() {
			return Identity{}, errors.Wrap(errIdentify, errors.ErrAuthentication)
		}

		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	case APIKey:
		_, err := svc.keys.Retrieve(context.TODO(), key.IssuerID, key.ID)
		if err != nil {
			return Identity{}, errors.ErrAuthentication
		}
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	default:
		return Identity{}, errors.ErrAuthentication
	}
}

func (svc service) tmpKey(duration time.Duration, key Key) (Key, string, error) {
	key.ExpiresAt = key.IssuedAt.Add(duration)
	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueTmp, err)
	}

	return key, secret, nil
}

func (svc service) login(token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", "", err
	}
	// Only login key token is valid for login.
	if key.Type != LoginKey || key.IssuerID == "" {
		return "", "", errors.ErrAuthentication
	}

	return key.IssuerID, key.Subject, nil
}

func getTimestamp() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (svc service) Backup(ctx context.Context, token string) (Backup, error) {
	if err := svc.isAdmin(ctx, token); err != nil {
		return Backup{}, err
	}

	orgs, err := svc.orgs.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	mrs, err := svc.memberships.BackupAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	backup := Backup{
		Orgs:           orgs,
		OrgMemberships: mrs,
	}

	return backup, nil
}

func (svc service) Restore(ctx context.Context, token string, backup Backup) error {
	if err := svc.isAdmin(ctx, token); err != nil {
		return err
	}

	if err := svc.orgs.Save(ctx, backup.Orgs...); err != nil {
		return err
	}

	if err := svc.memberships.Save(ctx, backup.OrgMemberships...); err != nil {
		return err
	}

	return nil
}

func (svc service) isAdmin(ctx context.Context, token string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}

	role, err := svc.roles.RetrieveRole(ctx, user.ID)
	if err != nil {
		return err
	}

	if role != RoleAdmin && role != RoleRootAdmin {
		return errors.ErrAuthorization
	}

	return nil
}
