// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

const (
	// Event prefixes
	thingPrefix           = "thing."
	profilePrefix         = "profile."
	groupPrefix           = "group."
	groupMembershipPrefix = "group_membership."
	orgPrefix             = "org."
	orgMembershipPrefix   = "org_membership."
	orgInvitePrefix       = "org_invite."
	apiKeyPrefix          = "api_key."
	sessionPrefix         = "session."
	userPrefix            = "user."
	platformInvitePrefix  = "platform_invite."
	certPrefix            = "cert."
	webhookPrefix         = "webhook."
	notifierPrefix        = "notifier."
	mainfluxPrefix        = "mainflux."

	// Event operation strings
	ThingCreate                = thingPrefix + "create"
	ThingUpdate                = thingPrefix + "update"
	ThingUpdateGroupAndProfile = thingPrefix + "update_group_and_profile"
	ThingUpdateMetadata        = thingPrefix + "update_metadata"
	ThingUpdateExternalKey     = thingPrefix + "update_external_key"
	ThingRemoveExternalKey     = thingPrefix + "remove_external_key"
	ThingRemove                = thingPrefix + "remove"

	ProfileCreate = profilePrefix + "create"
	ProfileUpdate = profilePrefix + "update"
	ProfileRemove = profilePrefix + "remove"

	GroupCreate = groupPrefix + "create"
	GroupUpdate = groupPrefix + "update"
	GroupRemove = groupPrefix + "remove"

	GroupMembershipCreate = groupMembershipPrefix + "create"
	GroupMembershipUpdate = groupMembershipPrefix + "update"
	GroupMembershipRemove = groupMembershipPrefix + "remove"

	OrgCreate = orgPrefix + "create"
	OrgUpdate = orgPrefix + "update"
	OrgRemove = orgPrefix + "remove"

	OrgMembershipCreate = orgMembershipPrefix + "create"
	OrgMembershipUpdate = orgMembershipPrefix + "update"
	OrgMembershipRemove = orgMembershipPrefix + "remove"

	OrgInviteCreate  = orgInvitePrefix + "create"
	OrgInviteRevoke  = orgInvitePrefix + "revoke"
	OrgInviteAccept  = orgInvitePrefix + "accept"
	OrgInviteDecline = orgInvitePrefix + "decline"

	APIKeyIssue  = apiKeyPrefix + "issue"
	APIKeyRevoke = apiKeyPrefix + "revoke"

	SessionLogoutAll = sessionPrefix + "logout_all"

	UserRegister          = userPrefix + "register"
	UserLogin             = userPrefix + "login"
	UserLoginFail         = userPrefix + "login_fail"
	UserUpdate            = userPrefix + "update"
	UserChangePassword    = userPrefix + "change_password"
	UserResetPassword     = userPrefix + "reset_password"
	UserEnable            = userPrefix + "enable"
	UserDisable           = userPrefix + "disable"
	UserAssignAdminRole   = userPrefix + "assign_admin_role"
	UserUnassignAdminRole = userPrefix + "unassign_admin_role"
	UserImpersonate       = userPrefix + "impersonate"

	PlatformInviteCreate = platformInvitePrefix + "create"
	PlatformInviteRevoke = platformInvitePrefix + "revoke"

	CertIssue  = certPrefix + "issue"
	CertRotate = certPrefix + "rotate"
	CertRevoke = certPrefix + "revoke"
	CertRenew  = certPrefix + "renew"

	WebhookCreate = webhookPrefix + "create"
	WebhookUpdate = webhookPrefix + "update"
	WebhookRemove = webhookPrefix + "remove"

	NotifierCreate = notifierPrefix + "create"
	NotifierUpdate = notifierPrefix + "update"
	NotifierRemove = notifierPrefix + "remove"

	// General redis event map fields
	jwtIdentityUserID    = "jwt_identity_user_id"
	jwtIdentityUserEmail = "jwt_identity_user_email"

	jwtIdentityImpersonatorID    = "jwt_identity_impersonator_id"
	jwtIdentityImpersonatorEmail = "jwt_identity_impersonator_email"

	occurredAt = "occurred_at"

	evtOrgID   = "evt_org_id"
	evtGroupID = "evt_group_id"

	// Redis event streams
	ThingsStream       = mainfluxPrefix + "things"
	AuthStream         = mainfluxPrefix + "auth"
	UsersStream        = mainfluxPrefix + "users"
	CertsStream        = mainfluxPrefix + "certs"
	WebhooksStream     = mainfluxPrefix + "webhooks"
	SMTPNotifierStream = mainfluxPrefix + "smtp-notifier"
	SMPPNotifierStream = mainfluxPrefix + "smpp-notifier"
)

// redisEvent is the raw payload delivered on a Redis stream.
type redisEvent map[string]any

// operation returns the event's operation name, or an empty string if missing.
func (e redisEvent) operation() string {
	return e.field("operation", "")
}

// jwtUserIdentity returns the identity of the user that initiated the event.
func (e redisEvent) jwtUserIdentity() domain.Identity {
	var identity domain.Identity

	identity.ID, _ = e[jwtIdentityUserID].(string)
	identity.Email, _ = e[jwtIdentityUserEmail].(string)

	impersonatorID, _ := e[jwtIdentityImpersonatorID].(string)
	if impersonatorID != "" {
		impersonatorEmail, _ := e[jwtIdentityImpersonatorEmail].(string)
		identity.Impersonator = &domain.Identity{
			ID:    impersonatorID,
			Email: impersonatorEmail,
		}
	}

	return identity
}

// occurredAt returns the time at which the event occurred, or the zero time if missing or unparseable.
func (e redisEvent) occurredAt() time.Time {
	raw, ok := e[occurredAt].(string)
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// field returns the string value stored under key, or def if missing.
func (e redisEvent) field(key, def string) string {
	s, ok := e[key].(string)
	if !ok {
		return def
	}
	return s
}

// EventHandler reacts to a single event delivered by a Subscriber.
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

// cursorKey is the Redis key where a named subscriber stores its last
// processed message ID for the given stream.
func cursorKey(name, stream string) string {
	return fmt.Sprintf("mainflux:cursor:%s:%s", name, stream)
}
