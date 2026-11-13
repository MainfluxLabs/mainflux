// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webrtc"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc webrtc.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Get("/webrtc/ice-servers", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_ice_servers")(viewICEServersEndpoint(svc)),
		decodeViewICEServers,
		encodeResponse,
		opts...,
	))
	r.Post("/webrtc/things/:id/view", kithttp.NewServer(
		kitot.TraceServer(tracer, "view")(viewEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))
	r.Patch("/webrtc/sessions/:id/candidates", kithttp.NewServer(
		kitot.TraceServer(tracer, "add_viewer_candidate")(addCandidateEndpoint(svc)),
		decodeAddCandidate,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("webrtc"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeViewICEServers(_ context.Context, r *http.Request) (any, error) {
	return iceServersReq{token: apiutil.ExtractBearerToken(r)}, nil
}

func decodeView(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := viewReq{token: apiutil.ExtractBearerToken(r), thingID: bone.GetValue(r, apiutil.IDKey)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeAddCandidate(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := addCandidateReq{token: apiutil.ExtractBearerToken(r), sessionID: bone.GetValue(r, apiutil.IDKey)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", apiutil.ContentTypeJSON)

	if ar, ok := response.(apiutil.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	apiutil.EncodeError(err, w)
	apiutil.WriteErrorResponse(err, w)
}
