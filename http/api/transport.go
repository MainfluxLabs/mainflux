// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"io/ioutil"
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
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol    = "http"
	ctSenmlJSON = "application/senml+json"
	ctSenmlCBOR = "application/senml+cbor"
	ctJSON      = "application/json"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/messages", kithttp.NewServer(
		kitot.TraceServer(tracer, "publish")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/messages/*", kithttp.NewServer(
		kitot.TraceServer(tracer, "publish")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("http"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeRequest(ctx context.Context, r *http.Request) (any, error) {
	ct := r.Header.Get("Content-Type")
	if !strings.Contains(ct, ctSenmlJSON) &&
		!strings.Contains(ct, ctJSON) &&
		!strings.Contains(ct, ctSenmlCBOR) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	subject, err := messaging.CreateSubtopic(r.URL.Path)
	if err != nil {
		return nil, err
	}

	var thingKey things.ThingKey
	_, pass, ok := r.BasicAuth()
	switch {
	case ok:
		thingKey = things.ThingKey{Type: things.KeyTypeInternal, Value: pass}
	case !ok:
		thingKey = things.ExtractThingKey(r)
	}

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, apiutil.ErrMalformedEntity
	}
	defer r.Body.Close()

	req := publishReq{
		msg: protomfx.Message{
			Protocol: protocol,
			Subtopic: subject,
			Payload:  payload,
			Created:  time.Now().UnixNano(),
		},
		ThingKey: thingKey,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
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
