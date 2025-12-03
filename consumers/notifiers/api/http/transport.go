// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	nameKey = "name"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc notifiers.Service, logger log.Logger) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/groups/:id/notifiers", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_notifiers")(createNotifiersEndpoint(svc)),
		decodeCreateNotifiers,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/notifiers", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_notifiers_by_group")(listNotifiersByGroupEndpoint(svc)),
		decodeListNotifiers,
		encodeResponse,
		opts...,
	))
	r.Post("/groups/:id/notifiers/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_notifiers_by_group")(listNotifiersByGroupEndpoint(svc)),
		decodeSearchNotifiers,
		encodeResponse,
		opts...,
	))

	r.Get("/notifiers/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_notifier")(viewNotifierEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/notifiers/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_notifier")(updateNotifierEndpoint(svc)),
		decodeUpdateNotifier,
		encodeResponse,
		opts...,
	))
	r.Patch("/notifiers", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_notifiers")(removeNotifiersEndpoint(svc)),
		decodeRemoveNotifiers,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("notifiers"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateNotifiers(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createNotifiersReq{token: apiutil.ExtractBearerToken(r), groupID: bone.GetValue(r, apiutil.IDKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Notifiers); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := notifierReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, apiutil.IDKey)}

	return req, nil
}

func decodeListNotifiers(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}
	pm.Name = n

	req := listNotifiersReq{
		token:        apiutil.ExtractBearerToken(r),
		id:           bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchNotifiers(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listNotifiersReq{
		token:        apiutil.ExtractBearerToken(r),
		id:           bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeUpdateNotifier(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateNotifierReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveNotifiers(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeNotifiersReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
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
	switch {
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
