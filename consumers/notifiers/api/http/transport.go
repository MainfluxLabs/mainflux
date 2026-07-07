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

const (
	nameKey    = "name"
	contactKey = "contact"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc notifiers.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	r.Post("/groups/:id/notifiers", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "create_notifiers"),
			withIdentity,
		)(createNotifiersEndpoint(svc)),
		decodeCreateNotifiers,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/notifiers", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_notifiers_by_group"),
			withIdentity,
		)(listNotifiersByGroupEndpoint(svc)),
		decodeListNotifiers,
		encodeResponse,
		opts...,
	))
	r.Post("/groups/:id/notifiers/search", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "search_notifiers_by_group"),
			withIdentity,
		)(listNotifiersByGroupEndpoint(svc)),
		decodeSearchNotifiers,
		encodeResponse,
		opts...,
	))

	r.Get("/notifiers/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_notifier"),
			withIdentity,
		)(viewNotifierEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/notifiers/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "update_notifier"),
			withIdentity,
		)(updateNotifierEndpoint(svc)),
		decodeUpdateNotifier,
		encodeResponse,
		opts...,
	))
	r.Patch("/notifiers", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_notifiers"),
			withIdentity,
		)(removeNotifiersEndpoint(svc)),
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := notifierReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, apiutil.IDKey)}

	return req, nil
}

func buildPageMetadata(r *http.Request) (notifiers.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return notifiers.PageMetadata{}, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return notifiers.PageMetadata{}, err
	}

	m, err := apiutil.ReadMetadataQuery(r, apiutil.MetadataKey, nil)
	if err != nil {
		return notifiers.PageMetadata{}, err
	}

	c, err := apiutil.ReadStringQuery(r, contactKey, "")
	if err != nil {
		return notifiers.PageMetadata{}, err
	}

	return notifiers.PageMetadata{
		Offset:   base.Offset,
		Limit:    base.Limit,
		Order:    base.Order,
		Dir:      base.Dir,
		Name:     n,
		Metadata: m,
		Contact:  c,
	}, nil
}

func buildPageMetadataFromBody(r *http.Request) (notifiers.PageMetadata, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return notifiers.PageMetadata{
			Offset: apiutil.DefOffset,
			Limit:  apiutil.DefLimit,
			Order:  apiutil.IDOrder,
			Dir:    apiutil.DescDir,
		}, nil
	}

	var pm notifiers.PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		return notifiers.PageMetadata{}, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	if pm.Limit == 0 {
		pm.Limit = apiutil.DefLimit
	}

	if pm.Offset == 0 {
		pm.Offset = apiutil.DefOffset
	}

	if pm.Order == "" {
		pm.Order = apiutil.IDOrder
	}

	if pm.Dir == "" {
		pm.Dir = apiutil.DescDir
	}

	return pm, nil
}

func decodeListNotifiers(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
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

func decodeSearchNotifiers(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadataFromBody(r)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
