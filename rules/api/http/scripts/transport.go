// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package scripts

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for script API endpoints.
func MakeHandler(svc rules.Service, ac domain.AuthClient, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScriptsByGroup(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	name, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")

	req := listScriptsByGroupReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: rules.PageMetadata{
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeScriptsReq{
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
	case errors.Contains(err, rules.ErrScriptSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
