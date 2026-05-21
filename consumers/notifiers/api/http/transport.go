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
	nameKey = "name"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc notifiers.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	newServer := func(name string, e endpoint.Endpoint, decodeFunc kithttp.DecodeRequestFunc) *kithttp.Server {
		e = withIdentity(e)
		e = kitot.TraceServer(tracer, name)(e)
		return kithttp.NewServer(e, decodeFunc, encodeResponse, opts...)
	}

	r.Post("/groups/:id/notifiers", newServer(
		"create_notifiers",
		createNotifiersEndpoint(svc),
		decodeCreateNotifiers,
	))
	r.Get("/groups/:id/notifiers", newServer(
		"list_notifiers_by_group",
		listNotifiersByGroupEndpoint(svc),
		decodeListNotifiers,
	))
	r.Post("/groups/:id/notifiers/search", newServer(
		"search_notifiers_by_group",
		listNotifiersByGroupEndpoint(svc),
		decodeSearchNotifiers,
	))

	r.Get("/notifiers/:id", newServer(
		"view_notifier",
		viewNotifierEndpoint(svc),
		decodeRequest,
	))
	r.Put("/notifiers/:id", newServer(
		"update_notifier",
		updateNotifierEndpoint(svc),
		decodeUpdateNotifier,
	))
	r.Patch("/notifiers", newServer(
		"remove_notifiers",
		removeNotifiersEndpoint(svc),
		decodeRemoveNotifiers,
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

	n, _ := apiutil.ReadStringQuery(r, nameKey, "")
	m, _ := apiutil.ReadMetadataQuery(r, apiutil.MetadataKey, nil)

	return notifiers.PageMetadata{
		Offset:   base.Offset,
		Limit:    base.Limit,
		Order:    base.Order,
		Dir:      base.Dir,
		Name:     n,
		Metadata: m,
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
