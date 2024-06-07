// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

type groupThingsReq struct {
	Things []string `json:"things"`
}

type groupChannelsReq struct {
	Channels []string `json:"channels"`
}

// UserPasswordReq contains old and new passwords
type UserPasswordReq struct {
	OldPassword string `json:"old_password,omitempty"`
	Password    string `json:"password,omitempty"`
}

// ConnectionIDs contains ID lists of things and channel to be connected
type ConnectionIDs struct {
	ChannelID string   `json:"channel_id"`
	ThingIDs  []string `json:"thing_ids"`
}

// deleteChannelsReq contains IDs of channels to be deleted
type deleteChannelsReq struct {
	ChannelIDs []string `json:"channel_ids"`
}

// deleteThingsReq contains IDs of things to be deleted
type deleteThingsReq struct {
	ThingIDs []string `json:"thing_ids"`
}

// deleteGroupsReq contains IDs of groups to be deleted
type deleteGroupsReq struct {
	GroupIDs []string `json:"group_ids"`
}

// deleteWebhooksReq contains IDs of webhooks to be deleted
type deleteWebhooksReq struct {
	WebhookIDs []string `json:"webhook_ids"`
}

// assignMembersReq contains org members to be assigned
type assignMembersReq struct {
	OrgMembers []OrgMember `json:"org_members"`
}

// unassignMembersReq contains IDs of members to be unassigned
type unassignMemberReq struct {
	MemberIDs []string `json:"member_ids"`
}

// updateMemberReq contains members to be updated
type updateMemberReq struct {
	OrgMembers []OrgMember `json:"org_members"`
}
