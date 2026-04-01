// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
)

func messagesHandshake(svc ws.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeMessageRequest(r)
		if err != nil {
			encodeError(w, err)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to upgrade connection to websocket: %s", err.Error()))
			return
		}
		req.conn = conn
		client := ws.NewClient(conn)

		if err := svc.Subscribe(context.Background(), req.ThingKey, req.subtopic, client); err != nil {
			req.conn.Close()
			return
		}

		logger.Debug("Successfully upgraded communication to WS")
		msgs := make(chan []byte)

		// Listen for messages and publish them to broker
		go processMessages(svc, req, msgs)
		go listen(conn, msgs)
	}
}

func thingCommandsHandshake(svc ws.Service) http.HandlerFunc {
	return commandsHandshake(svc, processThingCommands)
}

func groupCommandsHandshake(svc ws.Service) http.HandlerFunc {
	return commandsHandshake(svc, processGroupCommands)
}

func commandsHandshake(svc ws.Service, processFn func(ws.Service, cmdConnReq, <-chan []byte)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCommandRequest(r)
		if err != nil {
			encodeError(w, err)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to upgrade connection to websocket: %s", err.Error()))
			return
		}
		req.conn = conn
		msgs := make(chan []byte)
		go processFn(svc, req, msgs)
		go listen(conn, msgs)
	}
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

func listen(conn *websocket.Conn, msgs chan<- []byte) {
	for {
		_, payload, err := conn.ReadMessage()

		if websocket.IsUnexpectedCloseError(err) {
			logger.Debug(fmt.Sprintf("Closing WS connection: %s", err.Error()))
			close(msgs)
			return
		}

		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to read message: %s", err.Error()))
			close(msgs)
			return
		}

		msgs <- payload
	}
}

func processMessages(svc ws.Service, req getConnByKey, msgs <-chan []byte) {
	for payload := range msgs {
		svc.Publish(context.Background(), req.ThingKey, buildMessage(req.subtopic, payload))
	}
	if err := svc.Unsubscribe(context.Background(), req.ThingKey, req.subtopic); err != nil {
		req.conn.Close()
	}
}

func processThingCommands(svc ws.Service, req cmdConnReq, msgs <-chan []byte) {
	for payload := range msgs {
		m := buildMessage(req.subtopic, payload)
		switch {
		case req.token != "":
			svc.SendCommandToThing(context.Background(), req.token, req.id, m)
		default:
			svc.SendCommandToThingByKey(context.Background(), req.thingKey, req.id, m)
		}
	}
	req.conn.Close()
}

func processGroupCommands(svc ws.Service, req cmdConnReq, msgs <-chan []byte) {
	for payload := range msgs {
		m := buildMessage(req.subtopic, payload)
		switch {
		case req.token != "":
			svc.SendCommandToGroup(context.Background(), req.token, req.id, m)
		default:
			svc.SendCommandToGroupByKey(context.Background(), req.thingKey, req.id, m)
		}
	}
	req.conn.Close()
}

func buildMessage(subtopic string, payload []byte) protomfx.Message {
	return protomfx.Message{
		Subtopic: subtopic,
		Protocol: protocol,
		Payload:  payload,
		Created:  time.Now().UnixNano(),
	}
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
