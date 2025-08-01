// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	mux.Post("/orgs/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_groups")(createGroupsEndpoint(svc)),
		decodeCreateGroups,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group")(viewGroupEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/things/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_by_thing")(viewGroupByThingEndpoint(svc)),
		decodeViewByThing,
		encodeResponse,
		opts...,
	))

	mux.Get("/profiles/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_by_profile")(viewGroupByProfileEndpoint(svc)),
		decodeViewByProfile,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(listGroupsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups_by_org")(listGroupsByOrgEndpoint(svc)),
		decodeListByOrg,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/groups/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_groups_by_org")(backupGroupsByOrgEndpoint(svc)),
		decodeBackupByOrg,
		encodeFileResponse,
		opts...,
	))

	mux.Post("/orgs/:id/groups/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_groups_by_org")(restoreGroupsByOrgEndpoint(svc)),
		decodeRestoreByOrg,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_groups")(listGroupsEndpoint(svc)),
		decodeSearch,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:id/groups/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_groups_by_org")(listGroupsByOrgEndpoint(svc)),
		decodeSearchByOrg,
		encodeResponse,
		opts...,
	))

	mux.Put("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(updateGroupEndpoint(svc)),
		decodeUpdateGroup,
		encodeResponse,
		opts...,
	))

	mux.Delete("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_group")(removeGroupEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_groups")(removeGroupsEndpoint(svc)),
		decodeRemoveGroups,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeCreateGroups(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Groups); err != nil {
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

func decodeViewByProfile(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewByProfileReq{
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

func decodeSearch(_ context.Context, r *http.Request) (interface{}, error) {
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

func decodeSearchByOrg(_ context.Context, r *http.Request) (interface{}, error) {
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

func decodeUpdateGroup(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateGroupReq{
		id:    bone.GetValue(r, apiutil.IDKey),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveGroups(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeGroupsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRestoreByOrg(ctx context.Context, r *http.Request) (interface{}, error) {
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

	if err := json.Unmarshal(data, &req.Groups); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
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

func encodeFileResponse(_ context.Context, w http.ResponseWriter, response interface{}) (err error) {
	w.Header().Set("Content-Type", apiutil.ContentTypeOctetStream)

	if fr, ok := response.(viewFileRes); ok {
		for k, v := range fr.Headers() {
			w.Header().Set(k, v)
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fr.FileName))
		w.WriteHeader(fr.Code())

		if fr.Empty() {
			return nil
		}

		_, err := w.Write(fr.File)
		return err
	}

	return nil
}

func decodeBackupByOrg(_ context.Context, r *http.Request) (interface{}, error) {
	req := backupByOrgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
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
		err == apiutil.ErrInvalidDirection:
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
