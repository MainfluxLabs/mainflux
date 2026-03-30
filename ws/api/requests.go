// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/gorilla/websocket"
)

type getConnByKey struct {
	domain.ThingKey
	subtopic string
	conn     *websocket.Conn
}

type cmdConnReq struct {
	token    string
	thingKey things.ThingKey
	id       string
	subtopic string
	conn     *websocket.Conn
}
