// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/users"
)

type eventStore struct {
	users.Service
	auth               domain.AuthClient
	pub                events.Publisher
	emailVerifyEnabled bool
}

// NewEventStoreMiddleware returns a users service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc users.Service, auth domain.AuthClient, pub events.Publisher, emailVerifyEnabled bool) users.Service {
	return eventStore{
		Service:            svc,
		auth:               auth,
		pub:                pub,
		emailVerifyEnabled: emailVerifyEnabled,
	}
}

func (es eventStore) SelfRegister(ctx context.Context, user users.User, redirectPath string) (string, error) {
	res, err := es.Service.SelfRegister(ctx, user, redirectPath)
	if err != nil {
		return res, err
	}

	// With e-mail verification enabled, the account is created and published on VerifyEmail.
	if es.emailVerifyEnabled {
		return res, nil
	}

	es.publishSelfRegistered(ctx, res, user.Email, "")

	return res, nil
}

func (es eventStore) VerifyEmail(ctx context.Context, confirmationToken string) (string, error) {
	userID, err := es.Service.VerifyEmail(ctx, confirmationToken)
	if err != nil {
		return userID, err
	}

	// A failed lookup only leaves the e-mail out of the event.
	var email string
	if up, err := es.Service.ListUsersByIDs(ctx, []string{userID}, users.PageMetadata{}); err == nil && len(up.Users) > 0 {
		email = up.Users[0].Email
	}

	es.publishSelfRegistered(ctx, userID, email, "")

	return userID, nil
}

func (es eventStore) RegisterByInvite(ctx context.Context, user users.User, inviteID, orgInviteRedirectPath string) (string, error) {
	userID, err := es.Service.RegisterByInvite(ctx, user, inviteID, orgInviteRedirectPath)
	if err != nil {
		return userID, err
	}

	es.publishSelfRegistered(ctx, userID, user.Email, inviteID)

	return userID, nil
}

func (es eventStore) Register(ctx context.Context, token string, user users.User) (string, error) {
	userID, err := es.Service.Register(ctx, token, user)
	if err != nil {
		return userID, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserRegistered{
			ID:    userID,
			Email: user.Email,
		},
	})

	return userID, nil
}

func (es eventStore) Login(ctx context.Context, user users.User) (string, error) {
	token, err := es.Service.Login(ctx, user)
	// A failed user lookup is a database failure, not a failed login.
	if errors.Contains(err, errors.ErrAuthentication) && !errors.Contains(err, dbutil.ErrRetrieveEntity) {
		es.pub.Publish(ctx, events.Event{
			Action: events.UserLoginFailed{Email: user.Email},
		})
	}

	return token, err
}

func (es eventStore) OAuthCallback(ctx context.Context, data users.OAuthCallbackData) (users.OAuthCallbackResult, error) {
	res, err := es.Service.OAuthCallback(ctx, data)
	if err != nil {
		return res, err
	}

	if res.Registered {
		es.publishSelfRegistered(ctx, res.User.ID, res.User.Email, data.InviteID)
	}

	return res, nil
}

func (es eventStore) UpdateUser(ctx context.Context, token string, user users.User) error {
	if err := es.Service.UpdateUser(ctx, token, user); err != nil {
		return err
	}

	identity, _ := authn.IdentityFromCtx(ctx)

	es.pub.Publish(ctx, events.Event{
		Action: events.UserUpdated{
			ID:       identity.ID,
			Metadata: user.Metadata,
		},
	})

	return nil
}

func (es eventStore) ChangePassword(ctx context.Context, token, email, password, oldPassword string) error {
	identity, _ := authn.IdentityFromCtx(ctx)
	userID := identity.ID

	// A provided e-mail means a platform owner or admin changed another user's password.
	// Looked up first with errors ignored, so the response is always the service's own.
	if email != "" {
		userID = ""
		if us, err := es.Service.ListUsersByEmails(ctx, []string{email}); err == nil && len(us) > 0 {
			userID = us[0].ID
		}
	}

	if err := es.Service.ChangePassword(ctx, token, email, password, oldPassword); err != nil {
		return err
	}

	if userID == "" {
		return nil
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserPasswordChanged{ID: userID},
	})

	return nil
}

func (es eventStore) ResetPassword(ctx context.Context, resetToken, password string) error {
	// Identified first so a failed lookup can't fail a saved reset.
	identity, identifyErr := es.auth.IdentifyRecovery(ctx, resetToken)

	if err := es.Service.ResetPassword(ctx, resetToken, password); err != nil {
		return err
	}

	if identifyErr != nil {
		return nil
	}

	es.pub.Publish(ctx, events.Event{
		Action:          events.UserPasswordReset{ID: identity.ID},
		JWTUserIdentity: identity,
	})

	return nil
}

func (es eventStore) EnableUser(ctx context.Context, token, id string) error {
	if err := es.Service.EnableUser(ctx, token, id); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserEnabled{ID: id},
	})

	return nil
}

func (es eventStore) DisableUser(ctx context.Context, token, id string) error {
	if err := es.Service.DisableUser(ctx, token, id); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserDisabled{ID: id},
	})

	return nil
}

func (es eventStore) AssignAdminRole(ctx context.Context, token, id string) error {
	if err := es.Service.AssignAdminRole(ctx, token, id); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserAdminRoleAssigned{ID: id},
	})

	return nil
}

func (es eventStore) UnassignAdminRole(ctx context.Context, token, id string) error {
	if err := es.Service.UnassignAdminRole(ctx, token, id); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.UserAdminRoleUnassigned{ID: id},
	})

	return nil
}

func (es eventStore) CreatePlatformInvite(ctx context.Context, token, redirectPath, email string, orgInvite domain.OrgInvite) (users.PlatformInvite, error) {
	invite, err := es.Service.CreatePlatformInvite(ctx, token, redirectPath, email, orgInvite)
	if err != nil {
		return invite, err
	}

	// The org invite is recorded by auth.
	es.pub.Publish(ctx, events.Event{
		Action: events.PlatformInviteCreated{
			ID:           invite.ID,
			InviteeEmail: invite.InviteeEmail,
		},
	})

	return invite, nil
}

func (es eventStore) RevokePlatformInvite(ctx context.Context, token, inviteID string) error {
	if err := es.Service.RevokePlatformInvite(ctx, token, inviteID); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.PlatformInviteRevoked{ID: inviteID},
	})

	return nil
}

// publishSelfRegistered attributes the registration to the new user, as no one is logged in.
func (es eventStore) publishSelfRegistered(ctx context.Context, userID, email, inviteID string) {
	es.pub.Publish(ctx, events.Event{
		Action: events.UserRegistered{
			ID:       userID,
			Email:    email,
			InviteID: inviteID,
		},
		JWTUserIdentity: domain.Identity{
			ID:    userID,
			Email: email,
		},
	})
}
