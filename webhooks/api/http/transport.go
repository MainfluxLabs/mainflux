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
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc webhooks.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
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

	r.Post("/things/:id/webhooks", newServer(
		"create_webhooks",
		createWebhooksEndpoint(svc),
		decodeCreateWebhooks,
	))
	r.Get("/things/:id/webhooks", newServer(
		"list_webhooks_by_thing",
		listWebhooksByThingEndpoint(svc),
		decodeListThingWebhooks,
	))
	r.Get("/groups/:id/webhooks", newServer(
		"list_webhooks_by_group",
		listWebhooksByGroupEndpoint(svc),
		decodeListGroupWebhooks,
	))
	r.Post("/things/:id/webhooks/search", newServer(
		"search_webhooks_by_thing",
		listWebhooksByThingEndpoint(svc),
		decodeSearchThingWebhooks,
	))
	r.Post("/groups/:id/webhooks/search", newServer(
		"search_webhooks_by_group",
		listWebhooksByGroupEndpoint(svc),
		decodeSearchGroupWebhooks,
	))
	r.Get("/webhooks/:id", newServer(
		"view_webhook",
		viewWebhookEndpoint(svc),
		decodeRequest,
	))
	r.Put("/webhooks/:id", newServer(
		"update_webhook",
		updateWebhookEndpoint(svc),
		decodeUpdateWebhook,
	))
	r.Patch("/webhooks", newServer(
		"remove_webhooks",
		removeWebhooksEndpoint(svc),
		decodeRemoveWebhooks,
	))

	r.GetFunc("/health", mainflux.Health("webhooks"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateWebhooks(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createWebhooksReq{token: apiutil.ExtractBearerToken(r), thingID: bone.GetValue(r, apiutil.IDKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Webhooks); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := webhookReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, apiutil.IDKey)}

	return req, nil
}

func buildPageMetadata(r *http.Request) (webhooks.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return webhooks.PageMetadata{}, err
	}

	n, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	m, _ := apiutil.ReadMetadataQuery(r, apiutil.MetadataKey, nil)

	return webhooks.PageMetadata{
		Offset:   base.Offset,
		Limit:    base.Limit,
		Order:    base.Order,
		Dir:      base.Dir,
		Name:     n,
		Metadata: m,
	}, nil
}

func buildPageMetadataFromBody(r *http.Request) (webhooks.PageMetadata, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return webhooks.PageMetadata{
			Offset: apiutil.DefOffset,
			Limit:  apiutil.DefLimit,
			Order:  apiutil.IDOrder,
			Dir:    apiutil.DescDir,
		}, nil
	}

	var pm webhooks.PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		return webhooks.PageMetadata{}, errors.Wrap(errors.ErrMalformedEntity, err)
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

func decodeListGroupWebhooks(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listWebhooksByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListThingWebhooks(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listWebhooksByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchGroupWebhooks(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listWebhooksByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchThingWebhooks(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listWebhooksByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeUpdateWebhook(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateWebhookReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveWebhooks(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeWebhooksReq{
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
	case err == ErrInvalidUrl,
		err == ErrMissingWebhookID:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
