// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol            = "ws"
	readwriteBufferSize = 1024
)


var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  readwriteBufferSize,
		WriteBufferSize: readwriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	logger log.Logger
)

// MakeHandler returns http handler with websocket endpoints.
func MakeHandler(svc ws.Service, l log.Logger) http.Handler {
	logger = l

	mux := bone.New()
	mux.GetFunc("/messages", messagesHandshake(svc))
	mux.GetFunc("/messages/*", messagesHandshake(svc))
	mux.GetFunc("/things/:id/commands", thingCommandsHandshake(svc))
	mux.GetFunc("/things/:id/commands/*", thingCommandsHandshake(svc))
	mux.GetFunc("/groups/:id/commands", groupCommandsHandshake(svc))
	mux.GetFunc("/groups/:id/commands/*", groupCommandsHandshake(svc))
	mux.GetFunc("/version", mainflux.Health(protocol))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
