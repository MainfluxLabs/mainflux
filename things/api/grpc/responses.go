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
	profileID string
	thingID   string
	config    *protomfx.Config
}

type emptyRes struct {
	err error
}

type getGroupsByIDsRes struct {
	groups []*protomfx.Group
}

type configByThingIDRes struct {
	config *protomfx.Config
}

type groupIDByThingIDRes struct {
	groupID string
}
