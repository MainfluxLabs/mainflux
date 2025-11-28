// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	emailKey      = "email"
	statusKey     = "status"
	emailTokenKey = "token"
	stateKey      = "state"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, tracer opentracing.Tracer, logger logger.Logger, passwordRegex *regexp.Regexp) http.Handler {
	userPasswordRegex = passwordRegex

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux := bone.New()

	mux.Post("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "register")(registrationEndpoint(svc)),
		decodeRegisterUser,
		encodeResponse,
		opts...,
	))

	mux.Post("/register", kithttp.NewServer(
		kitot.TraceServer(tracer, "self_register")(selfRegistrationEndpoint(svc)),
		decodeSelfRegisterUser,
		encodeResponse,
		opts...,
	))

	mux.Post("/register/verify", kithttp.NewServer(
		kitot.TraceServer(tracer, "verify_email")(verifyEmailEndpoint(svc)),
		decodeVerifyEmail,
		encodeResponse,
		opts...,
	))

	mux.Post("/register/invite/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "register_by_invite")(inviteRegistrationEndpoint(svc)),
		decodePlatformInviteRegister,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/profile", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_profile")(viewProfileEndpoint(svc)),
		decodeViewProfile,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_user")(viewUserEndpoint(svc)),
		decodeViewUser,
		encodeResponse,
		opts...,
	))

	mux.Get("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_users")(listUsersEndpoint(svc)),
		decodeListUsers,
		encodeResponse,
		opts...,
	))

	mux.Post("/users/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_users")(listUsersEndpoint(svc)),
		decodeSearchUsers,
		encodeResponse,
		opts...,
	))

	mux.Put("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_user")(updateUserEndpoint(svc)),
		decodeUpdateUser,
		encodeResponse,
		opts...,
	))

	mux.Post("/password/reset-request", kithttp.NewServer(
		kitot.TraceServer(tracer, "res-req")(passwordResetRequestEndpoint(svc)),
		decodePasswordResetRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/password/reset", kithttp.NewServer(
		kitot.TraceServer(tracer, "reset")(passwordResetEndpoint(svc)),
		decodePasswordReset,
		encodeResponse,
		opts...,
	))

	mux.Patch("/password", kithttp.NewServer(
		kitot.TraceServer(tracer, "reset")(passwordChangeEndpoint(svc)),
		decodePasswordChange,
		encodeResponse,
		opts...,
	))

	mux.Post("/tokens", kithttp.NewServer(
		kitot.TraceServer(tracer, "login")(loginEndpoint(svc)),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	mux.Post("/users/:id/enable", kithttp.NewServer(
		kitot.TraceServer(tracer, "enable_user")(enableUserEndpoint(svc)),
		decodeChangeUserStatus,
		encodeResponse,
		opts...,
	))

	mux.Post("/users/:id/disable", kithttp.NewServer(
		kitot.TraceServer(tracer, "disable_user")(disableUserEndpoint(svc)),
		decodeChangeUserStatus,
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

	mux.Delete("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_platform_invite")(revokePlatformInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup")(backupEndpoint(svc)),
		decodeBackup,
		encodeResponse,
		opts...,
	))

	mux.Post("/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore")(restoreEndpoint(svc)),
		decodeRestore,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health("users"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeViewUser(_ context.Context, r *http.Request) (any, error) {
	req := viewUserReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeViewProfile(_ context.Context, r *http.Request) (any, error) {
	req := viewUserReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeListUsers(_ context.Context, r *http.Request) (any, error) {
	o, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	e, err := apiutil.ReadStringQuery(r, emailKey, "")
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, apiutil.MetadataKey, nil)
	if err != nil {
		return nil, err
	}

	s, err := apiutil.ReadStringQuery(r, statusKey, users.EnabledStatusKey)
	if err != nil {
		return nil, err
	}

	or, err := apiutil.ReadStringQuery(r, apiutil.OrderKey, apiutil.IDOrder)
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, apiutil.DirKey, apiutil.DescDir)
	if err != nil {
		return nil, err
	}

	req := listUsersReq{
		token:    apiutil.ExtractBearerToken(r),
		status:   s,
		offset:   o,
		limit:    l,
		email:    e,
		metadata: m,
		order:    or,
		dir:      d,
	}
	return req, nil
}

func decodeSearchUsers(_ context.Context, r *http.Request) (any, error) {
	req := listUsersReq{
		token:  apiutil.ExtractBearerToken(r),
		status: users.EnabledStatusKey,
		offset: apiutil.DefOffset,
		limit:  apiutil.DefLimit,
		order:  apiutil.IDOrder,
		dir:    apiutil.DescDir,
	}

	if r.Body == nil || r.ContentLength == 0 {
		return req, nil
	}

	var pm users.PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	if pm.Offset > 0 {
		req.offset = pm.Offset
	}

	if pm.Limit > 0 {
		req.limit = pm.Limit
	}

	if pm.Order != "" {
		req.order = pm.Order
	}

	if pm.Dir != "" {
		req.dir = pm.Dir
	}

	if pm.Status != "" {
		req.status = pm.Status
	}

	req.email = pm.Email
	req.metadata = pm.Metadata

	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (any, error) {
	req := updateUserReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}
	user.Email = strings.TrimSpace(user.Email)
	return userReq{user}, nil
}

func decodeRegisterUser(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	user.Email = strings.TrimSpace(user.Email)
	req := registerUserReq{
		user:  user,
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeSelfRegisterUser(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := selfRegisterUserReq{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeVerifyEmail(_ context.Context, r *http.Request) (any, error) {
	token, err := apiutil.ReadStringQuery(r, emailTokenKey, "")
	if err != nil {
		return verifyEmailReq{}, err
	}

	req := verifyEmailReq{
		emailToken: token,
	}

	return req, nil
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

func decodePasswordResetRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordChange(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := passwChangeReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeChangeUserStatus(_ context.Context, r *http.Request) (any, error) {
	req := changeUserStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeBackup(_ context.Context, r *http.Request) (any, error) {
	req := backupReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeRestore(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreReq{token: apiutil.ExtractBearerToken(r)}
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
