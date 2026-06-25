// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/audit"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	operationKey  = "operation"
	actionDataKey = "action_data"
	fromKey       = "from"
	toKey         = "to"
)

func MakeHandler(svc audit.Service, ac domain.AuthClient, tracer opentracing.Tracer, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	withIdentity := authn.IdentityMiddleware(ac, logger)

	mux := bone.New()

	mux.Get("/events", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_events"),
			withIdentity,
		)(listEventsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/events", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_events_by_org"),
			withIdentity,
		)(listEventsByOrgEndpoint(svc)),
		decodeListByOrg,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/events", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_events_by_group"),
			withIdentity,
		)(listEventsByGroupEndpoint(svc)),
		decodeListByGroup,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("audit"))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func decodeList(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listEventsReq{
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListByOrg(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listEventsByOrgReq{
		orgID:        bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listEventsByGroupReq{
		groupID:      bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func buildPageMetadata(r *http.Request) (audit.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return audit.PageMetadata{}, err
	}

	email, err := apiutil.ReadStringQuery(r, apiutil.EmailKey, "")
	if err != nil {
		return audit.PageMetadata{}, err
	}

	operation, err := apiutil.ReadStringQuery(r, operationKey, "")
	if err != nil {
		return audit.PageMetadata{}, err
	}

	data, err := apiutil.ReadMetadataQuery(r, actionDataKey, nil)
	if err != nil {
		return audit.PageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return audit.PageMetadata{}, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return audit.PageMetadata{}, err
	}

	pm := audit.PageMetadata{
		Offset:     base.Offset,
		Limit:      base.Limit,
		Order:      base.Order,
		Dir:        base.Dir,
		Email:      email,
		Operation:  operation,
		ActionData: data,
	}

	if from != 0 {
		pm.From = time.Unix(from, 0).UTC()
	}
	if to != 0 {
		pm.To = time.Unix(to, 0).UTC()
	}

	return pm, nil
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
