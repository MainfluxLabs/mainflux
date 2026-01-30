// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc things.Service, mux *bone.Mux, tracer opentracing.Tracer, logger log.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/profiles/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_things")(createThingsEndpoint(svc)),
		decodeCreateThings,
		encodeResponse,
		opts...,
	))

	mux.Get("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_thing")(viewThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/metadata", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_metadata_by_key")(viewMetadataByKeyEndpoint(svc)),
		decodeViewMetadata,
		encodeResponse,
		opts...,
	))

	mux.Get("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things")(listThingsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.Get("/profiles/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_profile")(listThingsByProfileEndpoint(svc)),
		decodeListByProfile,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_group")(listThingsByGroupEndpoint(svc)),
		decodeListByGroup,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_org")(listThingsByOrgEndpoint(svc)),
		decodeListByOrg,
		encodeResponse,
		opts...,
	))

	mux.Post("/things/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_things")(listThingsEndpoint(svc)),
		decodeSearch,
		encodeResponse,
		opts...,
	))

	mux.Post("/profiles/:id/things/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_things_by_profile")(listThingsByProfileEndpoint(svc)),
		decodeSearchByProfile,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups/:id/things/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_things_by_group")(listThingsByGroupEndpoint(svc)),
		decodeSearchByGroup,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:id/things/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_things_by_org")(listThingsByOrgEndpoint(svc)),
		decodeSearchByOrg,
		encodeResponse,
		opts...,
	))

	mux.Put("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_thing")(updateThingEndpoint(svc)),
		decodeUpdateThing,
		encodeResponse,
		opts...,
	))

	mux.Patch("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_thing_group_and_profile")(updateThingGroupAndProfileEndpoint(svc)),
		decodeUpdateThingGroupAndProfile,
		encodeResponse,
		opts...,
	))

	mux.Put("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_things_metadata")(updateThingsMetadataEndpoint(svc)),
		decodeUpdateThingsMetadata,
		encodeResponse,
		opts...,
	))

	mux.Delete("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_thing")(removeThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_things")(removeThingsEndpoint(svc)),
		decodeRemoveThings,
		encodeResponse,
		opts...,
	))

	mux.Patch("/things/:id/external-key", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_external_key")(updateExternalKeyEndpoint(svc)),
		decodeUpdateExternalKey,
		encodeResponse,
		opts...,
	))

	mux.Delete("/things/:id/external-key", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_external_key")(removeExternalKeyEndpoint(svc)),
		decodeRemoveExternalKey,
		encodeResponse,
		opts...,
	))

	mux.Post("/identify", kithttp.NewServer(
		kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
		decodeIdentify,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("things"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeCreateThings(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createThingsReq{
		token:     apiutil.ExtractBearerToken(r),
		profileID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Things); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateThing(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateThingReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateThingGroupAndProfile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateThingGroupAndProfileReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeViewMetadata(_ context.Context, r *http.Request) (any, error) {
	req := viewMetadataReq{
		ThingKey: things.ExtractThingKey(r),
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := resourceReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListByProfile(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listByProfileReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listByGroupReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListByOrg(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listByOrgReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearch(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchByProfile(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listByProfileReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listByGroupReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeSearchByOrg(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadataFromBody(r)
	if err != nil {
		return nil, err
	}

	req := listByOrgReq{
		id:           bone.GetValue(r, apiutil.IDKey),
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeRemoveThings(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeThingsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateThingsMetadata(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateThingsMetadataReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Things); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeIdentify(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := identifyReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateExternalKey(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateExternalKeyReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveExternalKey(_ context.Context, r *http.Request) (any, error) {
	req := resourceReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
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
	case errors.Contains(err, dbutil.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
