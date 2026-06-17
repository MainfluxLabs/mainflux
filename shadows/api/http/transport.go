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
	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc shadows.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	r.Put("/things/:id/shadows", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "update_desired_state"),
			withIdentity,
		)(updateDesiredStateEndpoint(svc)),
		decodeUpdateDesiredState,
		encodeResponse,
		opts...,
	))
	r.Get("/things/:id/shadows", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_shadow"),
			withIdentity,
		)(viewShadowEndpoint(svc)),
		decodeShadowReq,
		encodeResponse,
		opts...,
	))
	r.Delete("/things/:id/shadows", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_shadow"),
			withIdentity,
		)(removeShadowEndpoint(svc)),
		decodeShadowReq,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("shadows"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeUpdateDesiredState(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateDesiredStateReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeShadowReq(_ context.Context, r *http.Request) (any, error) {
	req := shadowReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
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
	case errors.Contains(err, shadows.ErrShadowNotFound):
		w.WriteHeader(http.StatusNotFound)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
