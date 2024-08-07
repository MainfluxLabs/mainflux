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
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/webhooks"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	idKey       = "id"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc webhooks.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/groups/:id/webhooks", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_webhooks")(createWebhooksEndpoint(svc)),
		decodeCreateWebhooks,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/webhooks", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_webhooks_by_group")(listWebhooksByGroupEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Get("/webhooks/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_webhook")(viewWebhookEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/webhooks/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_webhook")(updateWebhookEndpoint(svc)),
		decodeUpdateWebhook,
		encodeResponse,
		opts...,
	))
	r.Patch("/groups/:id/webhooks", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_webhooks")(removeWebhooksEndpoint(svc)),
		decodeRemoveWebhooks,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("webhooks"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateWebhooks(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createWebhooksReq{token: apiutil.ExtractBearerToken(r), groupID: bone.GetValue(r, idKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Webhooks); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := webhookReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, idKey)}

	return req, nil
}

func decodeUpdateWebhook(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateWebhookReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveWebhooks(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeWebhooksReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

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
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken,
		err == apiutil.ErrBearerKey:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrInvalidIDFormat,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingID,
		err == ErrInvalidUrl:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, readers.ErrReadMessages),
		errors.Contains(err, errors.ErrCreateEntity):
		w.WriteHeader(http.StatusInternalServerError)
	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrRetrieveEntity):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
