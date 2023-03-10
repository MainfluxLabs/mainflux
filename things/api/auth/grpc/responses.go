// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux"
)

type identityRes struct {
	id string
}

type emptyRes struct {
	err error
}

type getThingsByIDsRes struct {
	things []*mainflux.Thing
}
