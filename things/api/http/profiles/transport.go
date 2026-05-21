// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

	mux.Post("/groups/:id/profiles", newServer(
		"create_profiles",
		createProfilesEndpoint(svc),
		decodeCreateProfiles,
	))
	mux.Get("/profiles/:id", newServer(
		"view_profile",
		viewProfileEndpoint(svc),
		decodeRequest,
	))
	mux.Get("/things/:id/profiles", newServer(
		"view_profile_by_thing",
		viewProfileByThingEndpoint(svc),
		decodeViewByThing,
	))
	mux.Get("/profiles", newServer(
		"list_profiles",
		listProfilesEndpoint(svc),
		decodeList,
	))
	mux.Get("/groups/:id/profiles", newServer(
		"list_profiles_by_group",
		listProfilesByGroupEndpoint(svc),
		decodeListByGroup,
	))
	mux.Get("/orgs/:id/profiles", newServer(
		"list_profiles_by_org",
		listProfilesByOrgEndpoint(svc),
		decodeListByOrg,
	))
	mux.Post("/profiles/search", newServer(
		"search_profiles",
		listProfilesEndpoint(svc),
		decodeSearch,
	))
	mux.Post("/groups/:id/profiles/search", newServer(
		"search_profiles_by_group",
		listProfilesByGroupEndpoint(svc),
		decodeSearchByGroup,
	))
	mux.Post("/orgs/:id/profiles/search", newServer(
		"search_profiles_by_org",
		listProfilesByOrgEndpoint(svc),
		decodeSearchByOrg,
	))
	mux.Put("/profiles/:id", newServer(
		"update_profile",
		updateProfileEndpoint(svc),
		decodeUpdateProfile,
	))
	mux.Delete("/profiles/:id", newServer(
		"remove_profile",
		removeProfileEndpoint(svc),
		decodeRequest,
	))
	mux.Patch("/profiles", newServer(
		"remove_profiles",
		removeProfilesEndpoint(svc),
		decodeRemoveProfiles,
	))

	return mux
}

func decodeCreateProfiles(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createProfilesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Profiles); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateProfile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateProfileReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveProfiles(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeProfilesReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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

func decodeViewByThing(_ context.Context, r *http.Request) (any, error) {
	req := viewByThingReq{
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
	case errors.Contains(err, things.ErrProfileAssigned):
		w.WriteHeader(http.StatusConflict)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
