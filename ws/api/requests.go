// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/gorilla/websocket"
)

type getConnByKey struct {
	things.ThingKey
	subtopic string
	conn     *websocket.Conn
}
