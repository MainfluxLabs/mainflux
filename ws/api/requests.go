// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/gorilla/websocket"

type getConnByKey struct {
	thingKey string
	subtopic string
	conn     *websocket.Conn
}
