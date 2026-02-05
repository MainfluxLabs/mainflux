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

	var thingKey things.ThingKey
	_, pass, ok := r.BasicAuth()
	switch {
	case ok:
		thingKey = things.ThingKey{Type: things.KeyTypeInternal, Value: pass}
	case !ok:
		thingKey = things.ExtractThingKey(r)
	}

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

	req := thingCommandReq{
		cmdReq{
			token: apiutil.ExtractBearerToken(r),
			id:    id,
			msg: protomfx.Message{
				Subtopic: subtopic,
				Protocol: protocol,
				Payload:  payload,
				Created:  time.Now().UnixNano(),
			},
		},
	}

	return req, nil
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

	req := groupCommandReq{
		cmdReq{
			token: apiutil.ExtractBearerToken(r),
			id:    id,
			msg: protomfx.Message{
				Subtopic: subtopic,
				Protocol: protocol,
				Payload:  payload,
				Created:  time.Now().UnixNano(),
			},
		},
	}

	return req, nil
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
	if fullPath == basePath {
		return ""
	}

	if strings.HasPrefix(fullPath, basePath+"/") {
		return strings.TrimPrefix(fullPath, basePath+"/")
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
