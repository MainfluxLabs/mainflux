// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package orgs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
	"github.com/MainfluxLabs/mainflux/uiconfigs/api/http/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc uiconfigs.Service, mux *bone.Mux, logger log.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Get("/orgs/:id/configs", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org_config")(viewOrgConfigEndpoint(svc)),
		decodeViewOrgConfig,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/configs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_all_orgs_configs")(listOrgsConfigsEndpoint(svc)),
		decodeListOrgsConfigs,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id/configs", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_org_config")(updateOrgConfigEndpoint(svc)),
		decodeUpdateOrgConfig,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeViewOrgConfig(_ context.Context, r *http.Request) (any, error) {
	req := viewOrgConfigReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeListOrgsConfigs(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listOrgsConfigsReq{
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeUpdateOrgConfig(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateOrgConfigReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	case err == things.ErrMissingConfig:
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
