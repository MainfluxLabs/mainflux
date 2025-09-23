// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/ws"
	"github.com/gorilla/websocket"
)

func handshake(svc ws.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeRequest(r)
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

		if err := svc.Subscribe(context.Background(), req.ThingKey.Type, req.ThingKey.Key, req.subtopic, client); err != nil {
			req.conn.Close()
			return
		}

		logger.Debug("Successfully upgraded communication to WS")
		msgs := make(chan []byte)

		// Listen for messages and publish them to broker
		go process(svc, req, msgs)
		go listen(conn, msgs)
	}
}

func decodeRequest(r *http.Request) (getConnByKey, error) {
	authKey := apiutil.ExtractThingKey(r)
	if authKey.Key == "" || authKey.Type == "" {
		// Thing key not present in Authorization HTTP header, attempt to extract it from query params
		// TODO: handle extracting thing key from query params. perhaps from separate ones:
		// one for the key and one for the type? or maybe one param with a prefix denoting the type of key?
	}

	if err := authKey.Validate(); err != nil {
		return getConnByKey{}, err
	}

	req := getConnByKey{
		ThingKey: authKey,
	}

	subject, err := messaging.CreateSubject(r.RequestURI)
	if err != nil {
		return getConnByKey{}, err
	}

	req.subtopic = subject

	return req, nil
}

func listen(conn *websocket.Conn, msgs chan<- []byte) {
	for {
		// Listen for message from the client, and push them to the msgs profile
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

func process(svc ws.Service, req getConnByKey, msgs <-chan []byte) {
	for msg := range msgs {
		m := protomfx.Message{
			Subtopic: req.subtopic,
			Protocol: "websocket",
			Payload:  msg,
			Created:  time.Now().UnixNano(),
		}
		svc.Publish(context.Background(), req.ThingKey.Type, req.ThingKey.Key, m)
	}
	if err := svc.Unsubscribe(context.Background(), req.ThingKey.Type, req.ThingKey.Key, req.subtopic); err != nil {
		req.conn.Close()
	}
}

func encodeError(w http.ResponseWriter, err error) {
	var statusCode int

	switch err {
	case apiutil.ErrBearerKey,
		apiutil.ErrInvalidThingKeyType:
		statusCode = http.StatusUnauthorized
	case ws.ErrEmptyTopic:
		statusCode = http.StatusBadRequest
	case errUnauthorizedAccess:
		statusCode = http.StatusForbidden
	case messaging.ErrMalformedSubtopic, apiutil.ErrMalformedEntity:
		statusCode = http.StatusBadRequest
	default:
		statusCode = http.StatusNotFound
	}
	logger.Warn(fmt.Sprintf("Failed to authorize: %s", err.Error()))
	w.WriteHeader(statusCode)
}
