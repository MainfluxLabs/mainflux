// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
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

func decodeMessageRequest(r *http.Request) (getConnByKey, error) {
	tk := extractThingKey(r)
	if err := apiutil.ValidateThingKey(tk); err != nil {
		return getConnByKey{}, err
	}

	var subtopicPath string
	if idx := strings.Index(r.URL.Path, "/messages/"); idx >= 0 {
		subtopicPath = r.URL.Path[idx+len("/messages/"):]
	}
	subtopic, err := messaging.NormalizeSubtopic(subtopicPath)
	if err != nil {
		return getConnByKey{}, err
	}

	return getConnByKey{ThingKey: tk, subtopic: subtopic}, nil
}

func decodeCommandRequest(r *http.Request) (cmdConnReq, error) {
	id := bone.GetValue(r, apiutil.IDKey)
	if id == "" {
		return cmdConnReq{}, errors.ErrMalformedEntity
	}

	var subtopicPath string
	if idx := strings.Index(r.URL.Path, "/commands/"); idx >= 0 {
		subtopicPath = r.URL.Path[idx+len("/commands/"):]
	}
	subtopic, err := messaging.NormalizeSubtopic(subtopicPath)
	if err != nil {
		return cmdConnReq{}, err
	}

	req := cmdConnReq{id: id, subtopic: subtopic}

	tk := extractThingKey(r)
	if apiutil.ValidateThingKey(tk) == nil {
		req.thingKey = tk
		return req, nil
	}

	if token := extractBearerToken(r); token != "" {
		req.token = token
		return req, nil
	}

	return cmdConnReq{}, apiutil.ErrMissingAuth
}

// extractThingKey retrieves a ThingKey from the Authorization header,
// falling back to the key and keyType query parameters.
func extractThingKey(r *http.Request) domain.ThingKey {
	if tk := apiutil.ExtractThingKey(r); tk.Value != "" {
		return tk
	}

	queryKey := bone.GetQuery(r, "key")
	queryKeyType := bone.GetQuery(r, "keyType")
	if len(queryKey) > 0 && len(queryKeyType) > 0 {
		return domain.ThingKey{Value: queryKey[0], Type: queryKeyType[0]}
	}

	return domain.ThingKey{}
}

// extractBearerToken retrieves a bearer token from the Authorization header,
// falling back to the token query parameter for browser WS clients that cannot set headers.
func extractBearerToken(r *http.Request) string {
	if token := apiutil.ExtractBearerToken(r); token != "" {
		return token
	}

	if qt := bone.GetQuery(r, "token"); len(qt) > 0 {
		return qt[0]
	}

	return ""
}

func encodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Contains(err, messaging.ErrMalformedSubtopic):
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
