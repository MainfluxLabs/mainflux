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
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	idKey       = "id"
	offsetKey   = "offset"
	limitKey    = "limit"
	orderKey    = "order"
	dirKey      = "dir"
	defOffset   = 0
	defLimit    = 10
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

func decodeCreateNotifiers(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createNotifiersReq{token: apiutil.ExtractBearerToken(r), groupID: bone.GetValue(r, idKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Notifiers); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := notifierReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, idKey)}

	return req, nil
}

func decodeListNotifiers(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	or, err := apiutil.ReadStringQuery(r, orderKey, "")
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, dirKey, "")
	if err != nil {
		return nil, err
	}

	req := listNotifiersReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
		pageMetadata: things.PageMetadata{
			Offset: o,
			Limit:  l,
			Order:  or,
			Dir:    d,
		},
	}

	return req, nil
}

func decodeUpdateNotifier(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateNotifierReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveNotifiers(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
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
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrInvalidContact:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrRetrieveEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
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
