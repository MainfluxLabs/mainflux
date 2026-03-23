// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import "context"

// Client specifies the interface that things gRPC client implementations must fulfill.
type Client interface {
	GetPubConfigByKey(ctx context.Context, key ThingKey) (PubConfigInfo, error)
	GetConfigByThing(ctx context.Context, thingID string) (Config, error)
	CanUserAccessThing(ctx context.Context, ar UserAccessReq) error
	CanUserAccessProfile(ctx context.Context, ar UserAccessReq) error
	CanUserAccessGroup(ctx context.Context, ar UserAccessReq) error
	CanThingAccessGroup(ctx context.Context, ar ThingAccessReq) error
	Identify(ctx context.Context, key ThingKey) (string, error)
	GetGroupIDByThing(ctx context.Context, thingID string) (string, error)
	GetGroupIDByProfile(ctx context.Context, profileID string) (string, error)
	GetGroupIDsByOrg(ctx context.Context, ar OrgAccessReq) ([]string, error)
	GetThingIDsByProfile(ctx context.Context, profileID string) ([]string, error)
	CreateGroupMemberships(ctx context.Context, memberships ...GroupMembership) error
	GetGroup(ctx context.Context, groupID string) (Group, error)
	GetKeyByThingID(ctx context.Context, thingID string) (ThingKey, error)
}
