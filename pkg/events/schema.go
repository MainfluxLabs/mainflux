// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

// EventAction represents an action occurring on the platform.
type EventAction interface {
	// Encode returns a map representation of the event action.
	Encode() map[string]any

	// Operation returns the string representation of the event action operation.
	Operation() string
}

// Event is the type representing a platform event.
type Event struct {
	// The specific EventAction that occurred.
	Action EventAction
	// The identity of the user whose action triggered the event.
	JWTUserIdentity domain.Identity
	// The ID of the organization to which the event belongs. May be empty.
	OrgID string
	// The ID of the group to which the event belongs. May be empty.
	GroupID    string
	OccurredAt time.Time
}

// Encode turns an Event into a redisEvent
func (e Event) Encode() redisEvent {
	re := redisEvent(e.Action.Encode())
	re["operation"] = e.Action.Operation()

	if e.JWTUserIdentity.ID != "" {
		re[jwtIdentityUserID] = e.JWTUserIdentity.ID
		re[jwtIdentityUserEmail] = e.JWTUserIdentity.Email
	}

	if e.JWTUserIdentity.Impersonator != nil {
		re[jwtIdentityImpersonatorID] = e.JWTUserIdentity.Impersonator.ID
		re[jwtIdentityImpersonatorEmail] = e.JWTUserIdentity.Impersonator.Email
	}

	if !e.OccurredAt.IsZero() {
		re[occurredAt] = e.OccurredAt.UTC().Format(time.RFC3339Nano)
	}

	re[evtOrgID] = e.OrgID
	re[evtGroupID] = e.GroupID

	return re
}

// decodeEvent turns a raw RedisEvent into an Event.
func decodeEvent(re redisEvent) (Event, error) {
	var action EventAction

	op := re.operation()

	switch op {
	case ThingCreate:
		action = decodeThingCreated(re)
	case ThingUpdate:
		action = decodeThingUpdated(re)
	case ThingUpdateGroupAndProfile:
		action = decodeThingGroupAndProfileUpdated(re)
	case ThingRemove:
		action = decodeThingRemoved(re)
	case ProfileCreate:
		action = decodeProfileCreated(re)
	case ProfileUpdate:
		action = decodeProfileUpdated(re)
	case ProfileRemove:
		action = decodeProfileRemoved(re)
	case ThingUpdateMetadata:
		action = decodeThingMetadataUpdated(re)
	case ThingUpdateExternalKey:
		action = decodeThingExternalKeyUpdated(re)
	case ThingRemoveExternalKey:
		action = decodeThingExternalKeyRemoved(re)
	case GroupCreate:
		action = decodeGroupCreated(re)
	case GroupUpdate:
		action = decodeGroupUpdated(re)
	case GroupRemove:
		action = decodeGroupRemoved(re)
	case GroupMembershipCreate:
		action = decodeGroupMembershipCreated(re)
	case GroupMembershipUpdate:
		action = decodeGroupMembershipUpdated(re)
	case GroupMembershipRemove:
		action = decodeGroupMembershipRemoved(re)
	case OrgCreate:
		action = decodeOrgCreated(re)
	case OrgUpdate:
		action = decodeOrgUpdated(re)
	case OrgRemove:
		action = decodeOrgRemoved(re)
	case OrgMembershipCreate:
		action = decodeOrgMembershipCreated(re)
	case OrgMembershipUpdate:
		action = decodeOrgMembershipUpdated(re)
	case OrgMembershipRemove:
		action = decodeOrgMembershipRemoved(re)
	case OrgInviteCreate:
		action = decodeOrgInviteCreated(re)
	case OrgInviteRevoke:
		action = decodeOrgInviteRevoked(re)
	case OrgInviteAccept:
		action = decodeOrgInviteAccepted(re)
	case OrgInviteDecline:
		action = decodeOrgInviteDeclined(re)
	case APIKeyIssue:
		action = decodeAPIKeyIssued(re)
	case APIKeyRevoke:
		action = decodeAPIKeyRevoked(re)
	case SessionLogoutAll:
		action = decodeSessionLoggedOutAll(re)
	case UserRegister:
		action = decodeUserRegistered(re)
	case UserLogin:
		action = decodeUserLoggedIn(re)
	case UserLoginFail:
		action = decodeUserLoginFailed(re)
	case UserUpdate:
		action = decodeUserUpdated(re)
	case UserChangePassword:
		action = decodeUserPasswordChanged(re)
	case UserResetPassword:
		action = decodeUserPasswordReset(re)
	case UserEnable:
		action = decodeUserEnabled(re)
	case UserDisable:
		action = decodeUserDisabled(re)
	case UserAssignAdminRole:
		action = decodeUserAdminRoleAssigned(re)
	case UserUnassignAdminRole:
		action = decodeUserAdminRoleUnassigned(re)
	case UserImpersonate:
		action = decodeUserImpersonated(re)
	case PlatformInviteCreate:
		action = decodePlatformInviteCreated(re)
	case PlatformInviteRevoke:
		action = decodePlatformInviteRevoked(re)
	case WebhookCreate:
		action = decodeWebhookCreated(re)
	case WebhookUpdate:
		action = decodeWebhookUpdated(re)
	case WebhookRemove:
		action = decodeWebhookRemoved(re)
	default:
		return Event{}, fmt.Errorf("unknown event operation %s", op)
	}

	return Event{
		Action:          action,
		JWTUserIdentity: re.jwtUserIdentity(),
		OccurredAt:      re.occurredAt(),
		OrgID:           re.field(evtOrgID, ""),
		GroupID:         re.field(evtGroupID, ""),
	}, nil
}

// encodeDescribed encodes an entity that has a name, description and metadata.
func encodeDescribed(id, name, description string, metadata map[string]any) map[string]any {
	m := redisEvent{
		"id": id,
	}
	if name != "" {
		m["name"] = name
	}
	if description != "" {
		m["description"] = description
	}
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			m["metadata"] = string(b)
		}
	}
	return m
}
