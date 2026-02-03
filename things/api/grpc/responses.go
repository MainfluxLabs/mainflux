// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type identityRes struct {
	id string
}

type pubConfigByKeyRes struct {
	publisherID   string
	profileConfig *protomfx.Config
}

type configByThingRes struct {
	config *protomfx.Config
}

type emptyRes struct {
	err error
}

type groupIDRes struct {
	groupID string
}

type groupIDsRes struct {
	groupIDs []string
}

type thingIDsRes struct {
	thingIDs []string
}

type thingKeyRes struct {
	value   string
	keyType string
}

type groupRes struct {
	id    string
	orgID string
	name  string
}
