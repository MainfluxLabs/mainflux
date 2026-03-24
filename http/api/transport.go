// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	adapter "github.com/MainfluxLabs/mainflux/http"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol    = "http"
	ctSenmlJSON = "application/senml+json"
	ctSenmlCBOR = "application/senml+cbor"
	ctJSON      = "application/json"
	headerCT    = "Content-Type"

	messagesBasePath      = "/messages"
	thingCommandsBasePath = "/things/%s/commands"
	groupCommandsBasePath = "/groups/%s/commands"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/messages", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_message")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/messages/*", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_message")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:id/commands", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_command_to_thing")(sendCommandToThingEndpoint(svc)),
		decodeSendCommandToThing,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:id/commands/*", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_command_to_thing")(sendCommandToThingEndpoint(svc)),
		decodeSendCommandToThing,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/commands", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_command_to_group")(sendCommandToGroupEndpoint(svc)),
		decodeSendCommandByGroup,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/commands/*", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_command_to_group")(sendCommandToGroupEndpoint(svc)),
		decodeSendCommandByGroup,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("http"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	ct := r.Header.Get(headerCT)
	if !strings.Contains(ct, ctSenmlJSON) &&
		!strings.Contains(ct, ctJSON) &&
		!strings.Contains(ct, ctSenmlCBOR) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	thingKey := extractThingKey(r)

	payload, err := readPayload(r)
	if err != nil {
		return nil, err
	}

	subtopic := extractSubtopicFromPath(r.URL.Path, messagesBasePath)
	subtopic, err = messaging.NormalizeSubtopic(subtopic)
	if err != nil {
		return nil, err
	}

	req := publishReq{
		msg: protomfx.Message{
			Protocol: protocol,
			Subtopic: subtopic,
			Payload:  payload,
			Created:  time.Now().UnixNano(),
		},
		ThingKey: thingKey,
	}

	return req, nil
}

func decodeSendCommandToThing(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get(headerCT), ctJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	payload, err := readPayload(r)
	if err != nil {
		return nil, err
	}

	id := bone.GetValue(r, apiutil.IDKey)

	subtopic := extractSubtopicFromPath(r.URL.Path, fmt.Sprintf(thingCommandsBasePath, id))
	subtopic, err = messaging.NormalizeSubtopic(subtopic)
	if err != nil {
		return nil, err
	}

	req := cmdReq{
		id: id,
		msg: protomfx.Message{
			Subtopic: subtopic,
			Protocol: protocol,
			Payload:  payload,
			Created:  time.Now().UnixNano(),
		},
	}

	if tk := extractThingKey(r); tk.Value != "" {
		req.thingKey = tk
	} else {
		req.token = apiutil.ExtractBearerToken(r)
	}

	return thingCommandReq{req}, nil
}

func decodeSendCommandByGroup(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get(headerCT), ctJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	payload, err := readPayload(r)
	if err != nil {
		return nil, err
	}

	id := bone.GetValue(r, apiutil.IDKey)

	subtopic := extractSubtopicFromPath(r.URL.Path, fmt.Sprintf(groupCommandsBasePath, id))
	subtopic, err = messaging.NormalizeSubtopic(subtopic)
	if err != nil {
		return nil, err
	}

	req := cmdReq{
		id: id,
		msg: protomfx.Message{
			Subtopic: subtopic,
			Protocol: protocol,
			Payload:  payload,
			Created:  time.Now().UnixNano(),
		},
	}

	if tk := extractThingKey(r); tk.Value != "" {
		req.thingKey = tk
	} else {
		req.token = apiutil.ExtractBearerToken(r)
	}

	return groupCommandReq{req}, nil
}

func extractThingKey(r *http.Request) things.ThingKey {
	if _, pass, ok := r.BasicAuth(); ok {
		return things.ThingKey{Type: things.KeyTypeInternal, Value: pass}
	}
	return things.ExtractThingKey(r)
}

func readPayload(r *http.Request) ([]byte, error) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, apiutil.ErrMalformedEntity
	}
	defer r.Body.Close()

	return payload, nil
}

func extractSubtopicFromPath(fullPath string, basePath string) string {
	if sub, ok := strings.CutPrefix(fullPath, basePath+"/"); ok {
		return sub
	}
	return ""
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ any) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, messaging.ErrMalformedSubtopic):
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
