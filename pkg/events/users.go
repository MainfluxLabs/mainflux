// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import "encoding/json"

// UserRegistered signals that a new user account was created.
type UserRegistered struct {
	ID       string
	Email    string
	InviteID string
}

func (e UserRegistered) Operation() string {
	return UserRegister
}

func (e UserRegistered) Encode() map[string]any {
	m := redisEvent{
		"id":    e.ID,
		"email": e.Email,
	}
	if e.InviteID != "" {
		m["invite_id"] = e.InviteID
	}
	return m
}

func decodeUserRegistered(e redisEvent) UserRegistered {
	return UserRegistered{
		ID:       e.field("id", ""),
		Email:    e.field("email", ""),
		InviteID: e.field("invite_id", ""),
	}
}

// UserLoggedIn signals that a user started a new session.
type UserLoggedIn struct{}

func (e UserLoggedIn) Operation() string {
	return UserLogin
}

func (e UserLoggedIn) Encode() map[string]any {
	return redisEvent{}
}

func decodeUserLoggedIn(_ redisEvent) UserLoggedIn {
	return UserLoggedIn{}
}

// UserLoginFailed signals a login attempt with invalid credentials.
type UserLoginFailed struct {
	Email string
}

func (e UserLoginFailed) Operation() string {
	return UserLoginFail
}

func (e UserLoginFailed) Encode() map[string]any {
	return redisEvent{
		"email": e.Email,
	}
}

func decodeUserLoginFailed(e redisEvent) UserLoginFailed {
	return UserLoginFailed{Email: e.field("email", "")}
}

// UserUpdated signals that a user changed their metadata.
type UserUpdated struct {
	ID       string
	Metadata map[string]any
}

func (e UserUpdated) Operation() string {
	return UserUpdate
}

func (e UserUpdated) Encode() map[string]any {
	m := redisEvent{
		"id": e.ID,
	}
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}

func decodeUserUpdated(e redisEvent) UserUpdated {
	u := UserUpdated{ID: e.field("id", "")}
	if raw := e.field("metadata", ""); raw != "" {
		_ = json.Unmarshal([]byte(raw), &u.Metadata)
	}
	return u
}

// UserPasswordChanged signals that a user's password was changed by the user or a platform owner or admin.
type UserPasswordChanged struct {
	ID string
}

func (e UserPasswordChanged) Operation() string {
	return UserChangePassword
}

func (e UserPasswordChanged) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserPasswordChanged(e redisEvent) UserPasswordChanged {
	return UserPasswordChanged{ID: e.field("id", "")}
}

// UserPasswordReset signals that a user set a new password through the recovery flow.
type UserPasswordReset struct {
	ID string
}

func (e UserPasswordReset) Operation() string {
	return UserResetPassword
}

func (e UserPasswordReset) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserPasswordReset(e redisEvent) UserPasswordReset {
	return UserPasswordReset{ID: e.field("id", "")}
}

// UserEnabled signals that a platform owner or admin enabled a user.
type UserEnabled struct {
	ID string
}

func (e UserEnabled) Operation() string {
	return UserEnable
}

func (e UserEnabled) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserEnabled(e redisEvent) UserEnabled {
	return UserEnabled{ID: e.field("id", "")}
}

// UserDisabled signals that a platform owner or admin disabled a user.
type UserDisabled struct {
	ID string
}

func (e UserDisabled) Operation() string {
	return UserDisable
}

func (e UserDisabled) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserDisabled(e redisEvent) UserDisabled {
	return UserDisabled{ID: e.field("id", "")}
}

// UserAdminRoleAssigned signals that the platform owner assigned the platform admin role to a user.
type UserAdminRoleAssigned struct {
	ID string
}

func (e UserAdminRoleAssigned) Operation() string {
	return UserAssignAdminRole
}

func (e UserAdminRoleAssigned) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserAdminRoleAssigned(e redisEvent) UserAdminRoleAssigned {
	return UserAdminRoleAssigned{ID: e.field("id", "")}
}

// UserAdminRoleUnassigned signals that the platform owner removed the platform admin role from a user.
type UserAdminRoleUnassigned struct {
	ID string
}

func (e UserAdminRoleUnassigned) Operation() string {
	return UserUnassignAdminRole
}

func (e UserAdminRoleUnassigned) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodeUserAdminRoleUnassigned(e redisEvent) UserAdminRoleUnassigned {
	return UserAdminRoleUnassigned{ID: e.field("id", "")}
}

// PlatformInviteCreated signals that a person was invited to register on the platform.
type PlatformInviteCreated struct {
	ID           string
	InviteeEmail string
}

func (e PlatformInviteCreated) Operation() string {
	return PlatformInviteCreate
}

func (e PlatformInviteCreated) Encode() map[string]any {
	return redisEvent{
		"id":            e.ID,
		"invitee_email": e.InviteeEmail,
	}
}

func decodePlatformInviteCreated(e redisEvent) PlatformInviteCreated {
	return PlatformInviteCreated{
		ID:           e.field("id", ""),
		InviteeEmail: e.field("invitee_email", ""),
	}
}

// PlatformInviteRevoked signals that a platform owner or admin revoked a platform invite.
type PlatformInviteRevoked struct {
	ID string
}

func (e PlatformInviteRevoked) Operation() string {
	return PlatformInviteRevoke
}

func (e PlatformInviteRevoked) Encode() map[string]any {
	return redisEvent{
		"id": e.ID,
	}
}

func decodePlatformInviteRevoked(e redisEvent) PlatformInviteRevoked {
	return PlatformInviteRevoked{ID: e.field("id", "")}
}
