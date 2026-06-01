// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/pkg/domain"

type listJSONMessagesReq struct {
	thingKey domain.ThingKey
	pm       domain.JSONPageMetadata
}

type listSenMLMessagesReq struct {
	thingKey domain.ThingKey
	pm       domain.SenMLPageMetadata
}
