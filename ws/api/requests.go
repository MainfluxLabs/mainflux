// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	domainthings "github.com/MainfluxLabs/mainflux/pkg/domain/things"
	"github.com/gorilla/websocket"
)

type getConnByKey struct {
	domainthings.ThingKey
	subtopic string
	conn     *websocket.Conn
}
