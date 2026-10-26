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
	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const statusKey = "status"

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc ota.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	r.Post("/groups/:id/updates", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "create_update"),
			withIdentity,
		)(createUpdateEndpoint(svc)),
		decodeCreateUpdate,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/updates", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_updates_by_group"),
			withIdentity,
		)(listUpdatesEndpoint(svc)),
		decodeListUpdates,
		encodeResponse,
		opts...,
	))
	r.Get("/updates/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_update"),
			withIdentity,
		)(viewUpdateEndpoint(svc)),
		decodeUpdate,
		encodeResponse,
		opts...,
	))
	r.Delete("/updates/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_update"),
			withIdentity,
		)(removeUpdateEndpoint(svc)),
		decodeUpdate,
		encodeResponse,
		opts...,
	))
	r.Get("/updates/:id/things", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_thing_updates"),
			withIdentity,
		)(listThingUpdatesEndpoint(svc)),
		decodeListThingUpdates,
		encodeResponse,
		opts...,
	))
	r.Post("/updates/:id/retry", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "retry_failed"),
			withIdentity,
		)(retryFailedEndpoint(svc)),
		decodeUpdate,
		encodeResponse,
		opts...,
	))
	r.Get("/ota/pending", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_pending")(viewPendingEndpoint(svc)),
		decodeViewPending,
		encodeResponse,
		opts...,
	))
	r.Post("/ota/status", kithttp.NewServer(
		kitot.TraceServer(tracer, "report_status")(reportStatusEndpoint(svc)),
		decodeReportStatus,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("ota"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateUpdate(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createUpdateReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdate(_ context.Context, r *http.Request) (any, error) {
	req := updateReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeListUpdates(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listUpdatesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: ota.PageMetadata{
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return req, nil
}

func decodeListThingUpdates(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	s, err := apiutil.ReadStringQuery(r, statusKey, "")
	if err != nil {
		return nil, err
	}

	req := listThingUpdatesReq{
		token:    apiutil.ExtractBearerToken(r),
		updateID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: ota.PageMetadata{
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Status: s,
		},
	}

	return req, nil
}

func decodeViewPending(_ context.Context, r *http.Request) (any, error) {
	return viewPendingReq{key: apiutil.ExtractThingKey(r)}, nil
}

func decodeReportStatus(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := reportStatusReq{key: apiutil.ExtractThingKey(r)}
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
	switch {
	case errors.Contains(err, ErrMissingUpdateID),
		errors.Contains(err, ErrMissingFileName),
		errors.Contains(err, ErrMissingVersion),
		errors.Contains(err, ErrInvalidChecksum),
		errors.Contains(err, ErrInvalidFileSize),
		errors.Contains(err, ErrInvalidTimeout),
		errors.Contains(err, ota.ErrInvalidStatus),
		errors.Contains(err, ota.ErrMissingReportedVersion):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, ota.ErrUpdateFinished):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
