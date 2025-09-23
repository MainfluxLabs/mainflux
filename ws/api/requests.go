// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/gorilla/websocket"
)

type getConnByKey struct {
	apiutil.ThingKey
	subtopic string
	conn     *websocket.Conn
}
