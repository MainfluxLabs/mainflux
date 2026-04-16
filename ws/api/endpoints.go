// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
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
