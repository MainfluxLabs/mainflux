// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/runner"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	statusKey = "status"
	fromKey   = "from"
	toKey     = "to"
)

// MakeHandler returns a HTTP handler for Python script API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc pyscripts.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	mux := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	mux.Post("/groups/:id/scripts", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "create_scripts"),
			withIdentity,
		)(createScriptsEndpoint(svc)),
		decodeCreateScripts,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/scripts", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_scripts_by_group"),
			withIdentity,
		)(listScriptsByGroupEndpoint(svc)),
		decodeListScriptsByGroup,
		encodeResponse,
		opts...,
	))

	mux.Get("/scripts/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_script"),
			withIdentity,
		)(viewScriptEndpoint(svc)),
		decodeScriptReq,
		encodeResponse,
		opts...,
	))

	mux.Put("/scripts/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "update_script"),
			withIdentity,
		)(updateScriptEndpoint(svc)),
		decodeUpdateScript,
		encodeResponse,
		opts...,
	))

	mux.Patch("/scripts", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_scripts"),
			withIdentity,
		)(removeScriptsEndpoint(svc)),
		decodeRemoveScripts,
		encodeResponse,
		opts...,
	))

	mux.Post("/scripts/:id/run", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "run_script"),
			withIdentity,
		)(runScriptEndpoint(svc)),
		decodeRunScript,
		encodeResponse,
		opts...,
	))

	mux.Get("/scripts/:id/runs", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_runs_by_script"),
			withIdentity,
		)(listRunsByScriptEndpoint(svc)),
		decodeListRunsByScript,
		encodeResponse,
		opts...,
	))

	mux.Patch("/runs", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_runs"),
			withIdentity,
		)(removeRunsEndpoint(svc)),
		decodeRemoveRuns,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("pyscripts"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeCreateScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createScriptsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodySize)).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScriptsByGroup(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return nil, err
	}

	req := listScriptsByGroupReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pyscripts.PageMetadata{
			Offset: base.Offset,
			Limit:  base.Limit,
			Order:  base.Order,
			Dir:    base.Dir,
			Name:   name,
		},
	}

	return req, nil
}

func decodeScriptReq(_ context.Context, r *http.Request) (any, error) {
	req := scriptReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeUpdateScript(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateScriptReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodySize)).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeScriptsReq{token: apiutil.ExtractBearerToken(r)}

	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodySize)).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRunScript(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := runScriptReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	body := http.MaxBytesReader(nil, r.Body, MaxPayloadSize)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListRunsByScript(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	status, err := apiutil.ReadStringQuery(r, statusKey, "")
	if err != nil {
		return nil, err
	}

	fromMs, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	toMs, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	var from, to time.Time
	if fromMs > 0 {
		from = time.UnixMilli(fromMs)
	}
	if toMs > 0 {
		to = time.UnixMilli(toMs)
	}

	req := listRunsByScriptReq{
		token:    apiutil.ExtractBearerToken(r),
		scriptID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pyscripts.PageMetadata{
			Offset: base.Offset,
			Limit:  base.Limit,
			Order:  base.Order,
			Dir:    base.Dir,
			Status: status,
			From:   from,
			To:     to,
		},
	}

	return req, nil
}

func decodeRemoveRuns(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeRunsReq{token: apiutil.ExtractBearerToken(r)}

	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodySize)).Decode(&req); err != nil {
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
	w.Header().Set("Content-Type", apiutil.ContentTypeJSON)

	switch {
	case errors.Contains(err, pyscripts.ErrScriptSize),
		errors.Contains(err, pyscripts.ErrScriptSyntax):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, runner.ErrBusy):
		w.WriteHeader(http.StatusServiceUnavailable)
	case errors.Contains(err, runner.ErrWorkerFailed),
		errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
