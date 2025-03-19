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
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc things.Service, mux *bone.Mux, tracer opentracing.Tracer, logger log.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/groups/:id/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_profiles")(createProfilesEndpoint(svc)),
		decodeCreateProfiles,
		encodeResponse,
		opts...,
	))

	mux.Get("/profiles/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_profile")(viewProfileEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/things/:id/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_profile_by_thing")(viewProfileByThingEndpoint(svc)),
		decodeViewByThing,
		encodeResponse,
		opts...,
	))

	mux.Get("/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_profiles")(listProfilesEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_profiles_by_group")(listProfilesByGroupEndpoint(svc)),
		decodeListByGroup,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_profiles_by_org")(listProfilesByOrgEndpoint(svc)),
		decodeListByOrg,
		encodeResponse,
		opts...,
	))

	mux.Put("/profiles/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_profile")(updateProfileEndpoint(svc)),
		decodeUpdateProfile,
		encodeResponse,
		opts...,
	))

	mux.Delete("/profiles/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_profile")(removeProfileEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/profiles", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_profiles")(removeProfilesEndpoint(svc)),
		decodeRemoveProfiles,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeCreateProfiles(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createProfilesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Profiles); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateProfile(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateProfileReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveProfiles(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeProfilesReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := resourceReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeViewByThing(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewByThingReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
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

func decodeListByGroup(_ context.Context, r *http.Request) (interface{}, error) {
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

func decodeListByOrg(_ context.Context, r *http.Request) (interface{}, error) {
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

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
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
	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingThingID,
		err == apiutil.ErrMissingProfileID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrMissingOrgID,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrInvalidIDFormat:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
