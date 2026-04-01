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
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for script API endpoints.
func MakeHandler(svc rules.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/groups/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_scripts")(createScriptsEndpoint(svc)),
		decodeCreateScripts,
		encodeResponse,
		opts...,
	))

	mux.Get("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_scripts_by_thing")(listScriptsByThingEndpoint(svc)),
		decodeListScriptsByThing,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_scripts_by_group")(listScriptsByGroupEndpoint(svc)),
		decodeListScriptsByGroup,
		encodeResponse,
		opts...,
	))

	mux.Get("/scripts/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_thing_ids_by_script")(listThingIDsByScriptEndpoint(svc)),
		decodeScriptReq,
		encodeResponse,
		opts...,
	))

	mux.Get("/scripts/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_script")(viewScriptEndpoint(svc)),
		decodeScriptReq,
		encodeResponse,
		opts...,
	))

	mux.Put("/scripts/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_script")(updateScriptEndpoint(svc)),
		decodeUpdateScript,
		encodeResponse,
		opts...,
	))

	mux.Patch("/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_scripts")(removeScriptsEndpoint(svc)),
		decodeRemoveScripts,
		encodeResponse,
		opts...,
	))

	mux.Post("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_scripts")(assignScriptsEndpoint(svc)),
		decodeThingScripts,
		encodeResponse,
		opts...,
	))

	mux.Patch("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_scripts")(unassignScriptsEndpoint(svc)),
		decodeThingScripts,
		encodeResponse,
		opts...,
	))

	mux.Get("/things/:id/runs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_script_runs_by_thing")(listScriptRunsByThingEndpoint(svc)),
		decodeListScriptRunsByThing,
		encodeResponse,
		opts...,
	))

	mux.Patch("/runs", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_script_runs")(removeScriptRunsEndpoint(svc)),
		decodeRemoveScriptRuns,
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

func decodeListScriptsByThing(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	name, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")

	req := listScriptsByThingReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
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

func decodeThingScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := thingScriptsReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScriptRunsByThing(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}
	name, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")

	req := listScriptRunsByThingReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
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

func decodeRemoveScriptRuns(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeScriptRunsReq{
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
