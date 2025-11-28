// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

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

	mux.Post("/profiles/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_profiles")(listProfilesEndpoint(svc)),
		decodeSearch,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups/:id/profiles/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_profiles_by_group")(listProfilesByGroupEndpoint(svc)),
		decodeSearchByGroup,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:id/profiles/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_profiles_by_org")(listProfilesByOrgEndpoint(svc)),
		decodeSearchByOrg,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/profiles/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_profiles_by_org")(backupProfilesByOrgEndpoint(svc)),
		decodeBackupByOrg,
		apiutil.EncodeFileResponse,
		opts...,
	))

	mux.Post("/orgs/:id/profiles/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_profiles_by_org")(restoreProfilesByOrgEndpoint(svc)),
		decodeRestoreByOrg,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/profiles/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_profiles_by_group")(backupProfilesByGroupEndpoint(svc)),
		decodeBackupByGroup,
		apiutil.EncodeFileResponse,
		opts...,
	))

	mux.Post("/groups/:id/profiles/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_profiles_by_group")(restoreProfilesByGroupEndpoint(svc)),
		decodeRestoreByGroup,
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

func decodeCreateProfiles(_ context.Context, r *http.Request) (any, error) {
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

func decodeUpdateProfile(_ context.Context, r *http.Request) (any, error) {
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

func decodeRemoveProfiles(_ context.Context, r *http.Request) (any, error) {
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

func decodeBackupByGroup(_ context.Context, r *http.Request) (any, error) {
	req := backupByGroupReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	return req, nil
}

func decodeRestoreByGroup(ctx context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeOctetStream) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreByGroupReq{
		id:    bone.GetValue(r, apiutil.IDKey),
		token: apiutil.ExtractBearerToken(r),
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	if err := json.Unmarshal(data, &req.Profiles); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeBackupByOrg(_ context.Context, r *http.Request) (any, error) {
	req := backupByOrgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	return req, nil
}

func decodeRestoreByOrg(ctx context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeOctetStream) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreByOrgReq{
		id:    bone.GetValue(r, apiutil.IDKey),
		token: apiutil.ExtractBearerToken(r),
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	if err := json.Unmarshal(data, &req.Profiles); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
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
