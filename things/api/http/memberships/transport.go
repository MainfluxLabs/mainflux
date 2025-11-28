// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"context"
	"encoding/json"
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

const (
	emailKey = "email"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc things.Service, mux *bone.Mux, tracer opentracing.Tracer, logger log.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/groups/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_group_memberships")(createGroupMembershipsEndpoint(svc)),
		decodeGroupMemberships,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_group_memberships")(listGroupMembershipsEndpoint(svc)),
		decodeListGroupMemberships,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/memberships/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_group_memberships")(backupGroupMembershipsEndpoint(svc)),
		decodeBackupByGroup,
		apiutil.EncodeFileResponse,
		opts...,
	))

	mux.Post("/groups/:id/memberships/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_group_memberships")(restoreGroupMembershipsEndpoint(svc)),
		decodeRestoreByGroup,
		encodeResponse,
		opts...,
	))

	mux.Put("/groups/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group_memberships")(updateGroupMembershipsEndpoint(svc)),
		decodeGroupMemberships,
		encodeResponse,
		opts...,
	))

	mux.Patch("/groups/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_group_memberships")(removeGroupMembershipsEndpoint(svc)),
		decodeRemoveGroupMemberships,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeListGroupMemberships(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	e, err := apiutil.ReadStringQuery(r, emailKey, "")
	if err != nil {
		return nil, err
	}

	pm.Email = e

	req := listGroupMembershipsReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeGroupMemberships(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := groupMembershipsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveGroupMemberships(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeGroupMembershipsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
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

	if err := json.Unmarshal(data, &req.GroupMemberships); err != nil {
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
	case errors.Contains(err, things.ErrGroupMembershipExists):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
