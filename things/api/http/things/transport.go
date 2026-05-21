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
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc things.Service, ac domain.AuthClient, mux *bone.Mux, tracer opentracing.Tracer, logger log.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	withIdentity := authn.IdentityMiddleware(ac, logger)

	newServer := func(name string, e endpoint.Endpoint, decodeFunc kithttp.DecodeRequestFunc) *kithttp.Server {
		e = withIdentity(e)
		e = kitot.TraceServer(tracer, name)(e)
		return kithttp.NewServer(e, decodeFunc, encodeResponse, opts...)
	}

	mux.Post("/profiles/:id/things", newServer(
		"create_things",
		createThingsEndpoint(svc),
		decodeCreateThings,
	))

	mux.Get("/things/:id", newServer(
		"view_thing",
		viewThingEndpoint(svc),
		decodeRequest,
	))

	mux.Get("/metadata", newServer(
		"view_metadata_by_key",
		viewMetadataByKeyEndpoint(svc),
		decodeViewMetadata,
	))

	mux.Get("/things", newServer(
		"list_things",
		listThingsEndpoint(svc),
		decodeList,
	))

	mux.Get("/profiles/:id/things", newServer(
		"list_things_by_profile",
		listThingsByProfileEndpoint(svc),
		decodeListByProfile,
	))

	mux.Get("/groups/:id/things", newServer(
		"list_things_by_group",
		listThingsByGroupEndpoint(svc),
		decodeListByGroup,
	))

	mux.Get("/orgs/:id/things", newServer(
		"list_things_by_org",
		listThingsByOrgEndpoint(svc),
		decodeListByOrg,
	))

	mux.Post("/things/search", newServer(
		"search_things",
		listThingsEndpoint(svc),
		decodeSearch,
	))

	mux.Post("/profiles/:id/things/search", newServer(
		"search_things_by_profile",
		listThingsByProfileEndpoint(svc),
		decodeSearchByProfile,
	))

	mux.Post("/groups/:id/things/search", newServer(
		"search_things_by_group",
		listThingsByGroupEndpoint(svc),
		decodeSearchByGroup,
	))

	mux.Post("/orgs/:id/things/search", newServer(
		"search_things_by_org",
		listThingsByOrgEndpoint(svc),
		decodeSearchByOrg,
	))

	mux.Put("/things/:id", newServer(
		"update_thing",
		updateThingEndpoint(svc),
		decodeUpdateThing,
	))

	mux.Patch("/things/:id", newServer(
		"update_thing_group_and_profile",
		updateThingGroupAndProfileEndpoint(svc),
		decodeUpdateThingGroupAndProfile,
	))

	mux.Put("/things", newServer(
		"update_things_metadata",
		updateThingsMetadataEndpoint(svc),
		decodeUpdateThingsMetadata,
	))

	mux.Delete("/things/:id", newServer(
		"remove_thing",
		removeThingEndpoint(svc),
		decodeRequest,
	))

	mux.Patch("/things", newServer(
		"remove_things",
		removeThingsEndpoint(svc),
		decodeRemoveThings,
	))

	mux.Patch("/things/:id/external-key", newServer(
		"update_external_key",
		updateExternalKeyEndpoint(svc),
		decodeUpdateExternalKey,
	))

	mux.Delete("/things/:id/external-key", newServer(
		"remove_external_key",
		removeExternalKeyEndpoint(svc),
		decodeRemoveExternalKey,
	))

	mux.Post("/identify", newServer(
		"identify",
		identifyEndpoint(svc),
		decodeIdentify,
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeViewMetadata(_ context.Context, r *http.Request) (any, error) {
	req := viewMetadataReq{
		ThingKey: apiutil.ExtractThingKey(r),
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

func buildPageMetadata(r *http.Request) (things.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return things.PageMetadata{}, err
	}

	n, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	m, _ := apiutil.ReadMetadataQuery(r, apiutil.MetadataKey, nil)

	return things.PageMetadata{
		Offset:   base.Offset,
		Limit:    base.Limit,
		Order:    base.Order,
		Dir:      base.Dir,
		Name:     n,
		Metadata: m,
	}, nil
}

func buildPageMetadataFromBody(r *http.Request) (things.PageMetadata, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return things.PageMetadata{
			Offset: apiutil.DefOffset,
			Limit:  apiutil.DefLimit,
			Order:  apiutil.IDOrder,
			Dir:    apiutil.DescDir,
		}, nil
	}

	var pm things.PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		return things.PageMetadata{}, errors.Wrap(errors.ErrMalformedEntity, err)
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

func decodeList(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
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
	pm, err := buildPageMetadata(r)
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
	pm, err := buildPageMetadata(r)
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
	pm, err := buildPageMetadata(r)
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
	pm, err := buildPageMetadataFromBody(r)
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
	pm, err := buildPageMetadataFromBody(r)
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
	pm, err := buildPageMetadataFromBody(r)
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
	pm, err := buildPageMetadataFromBody(r)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeIdentify(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := identifyReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
