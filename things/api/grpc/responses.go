// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

type identityRes struct {
	id string
}

type pubConfigByKeyRes struct {
	domain.PubConfigInfo
}

type configByThingRes struct {
	config *domain.ProfileConfig
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
