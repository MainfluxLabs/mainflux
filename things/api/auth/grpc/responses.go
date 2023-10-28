// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux"

type identityRes struct {
	id string
}

type connByKeyRes struct {
	channelOD string
	thingID   string
}

type emptyRes struct {
	err error
}

type getGroupsByIDsRes struct {
	groups []*mainflux.Group
}
