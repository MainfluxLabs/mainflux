// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

// UserPasswordReq contains old and new passwords
type UserPasswordReq struct {
	OldPassword string `json:"old_password,omitempty"`
	Password    string `json:"password,omitempty"`
}

// deleteProfilesReq contains IDs of profiles to be deleted
type deleteProfilesReq struct {
	ProfileIDs []string `json:"profile_ids"`
}

// deleteThingsReq contains IDs of things to be deleted
type deleteThingsReq struct {
	ThingIDs []string `json:"thing_ids"`
}

// deleteGroupsReq contains IDs of groups to be deleted
type deleteGroupsReq struct {
	GroupIDs []string `json:"group_ids"`
}

// orgMembershipsReq contains org memberships to be created or updated
type orgMembershipsReq struct {
	OrgMemberships []OrgMembership `json:"org_memberships"`
}

// removeMembershipsReq contains IDs of members to be removed
type removeMembershipsReq struct {
	MemberIDs []string `json:"member_ids"`
}

// deleteWebhooksReq contains IDs of webhooks to be deleted
type deleteWebhooksReq struct {
	WebhookIDs []string `json:"webhook_ids"`
}
