// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/downlinks/api/http/backup"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
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
func MakeHandler(tracer opentracing.Tracer, svc downlinks.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/things/:id/downlinks", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_downlinks")(createDownlinksEndpoint(svc)),
		decodeCreateDownlinks,
		encodeResponse,
		opts...,
	))
	r.Get("/things/:id/downlinks", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_downlinks_by_thing")(listDownlinksByThingEndpoint(svc)),
		decodeListThingDownlinks,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/downlinks", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_downlinks_by_group")(listDownlinksByGroupEndpoint(svc)),
		decodeListDownlinks,
		encodeResponse,
		opts...,
	))
	r.Get("/downlinks/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_downlink")(viewDownlinkEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/downlinks/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_downlink")(updateDownlinkEndpoint(svc)),
		decodeUpdateDownlink,
		encodeResponse,
		opts...,
	))
	r.Patch("/downlinks", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_downlinks")(removeDownlinksEndpoint(svc)),
		decodeRemoveDownlinks,
		encodeResponse,
		opts...,
	))

	backup.MakeHandler(tracer, svc, r, logger)

	r.GetFunc("/health", mainflux.Health("downlinks"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateDownlinks(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createDownlinksReq{token: apiutil.ExtractBearerToken(r), thingID: bone.GetValue(r, idKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Downlinks); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListDownlinks(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listDownlinksReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, idKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListThingDownlinks(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listThingDownlinksReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, idKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := downlinkReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, idKey)}

	return req, nil
}

func decodeUpdateDownlink(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateDownlinkReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveDownlinks(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeDownlinksReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
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
	case err == ErrMissingID,
		err == ErrInvalidURL,
		err == ErrInvalidScheduler,
		err == ErrInvalidFilterParam,
		err == ErrMissingFilterFormat,
		err == ErrInvalidFilterInterval,
		err == ErrInvalidFilterValue:
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
