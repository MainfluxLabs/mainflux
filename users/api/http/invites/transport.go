// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

const stateKey = "state"

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger, passwordRegex *regexp.Regexp) *bone.Mux {
	userPasswordRegex = passwordRegex

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/register/invite/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "register_by_invite")(inviteRegistrationEndpoint(svc)),
		decodePlatformInviteRegister,
		encodeResponse,
		opts...,
	))

	mux.Post("/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_platform_invite")(createPlatformInviteEndpoint(svc)),
		decodeCreatePlatformInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_platform_invites")(listPlatformInvitesEndpoint(svc)),
		decodeListPlatformInvitesRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_platform_invite")(viewPlatformInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites/:id/public", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_platform_invite_public")(viewPlatformInvitePublicEndpoint(svc)),
		decodePublicInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_platform_invite")(revokePlatformInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodePlatformInviteRegister(_ context.Context, r *http.Request) (any, error) {
	req := registerByInviteReq{
		inviteID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := inviteReq{
		token:    apiutil.ExtractBearerToken(r),
		inviteID: bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodePublicInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := publicInviteReq{
		inviteID: bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeCreatePlatformInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := createPlatformInviteRequest{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListPlatformInvitesRequest(_ context.Context, r *http.Request) (any, error) {
	req := listPlatformInvitesRequest{
		token: apiutil.ExtractBearerToken(r),
	}

	pm, err := buildPageMetadataInvites(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

	return req, nil
}

func buildPageMetadataInvites(r *http.Request) (users.PageMetadataInvites, error) {
	pm := users.PageMetadataInvites{}

	apm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return users.PageMetadataInvites{}, err
	}

	pm.PageMetadata = apm

	state, err := apiutil.ReadStringQuery(r, stateKey, "")
	if err != nil {
		return users.PageMetadataInvites{}, err
	}

	pm.State = state

	return pm, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
	if ar, ok := response.(apiutil.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", apiutil.ContentTypeJSON)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, users.ErrPasswordFormat),
		errors.Contains(err, errors.ErrInvalidPassword),
		errors.Contains(err, users.ErrEmailVerificationExpired):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID),
		errors.Contains(err, users.ErrRecoveryToken):
		w.WriteHeader(http.StatusInternalServerError)
	case errors.Contains(err, users.ErrSelfRegisterDisabled):
		w.WriteHeader(http.StatusForbidden)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
