// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type identityRes struct {
	id string
}

type connByKeyRes struct {
	channelID string
	thingID   string
	profile   *protomfx.Profile
}

type emptyRes struct {
	err error
}

type getGroupsByIDsRes struct {
	groups []*protomfx.Group
}

type profileByThingIDRes struct {
	profile *protomfx.Profile
}

type groupIDByThingIDRes struct {
	groupID string
}
