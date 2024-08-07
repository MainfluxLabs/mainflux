// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
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
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	emailKey    = "email"
	metadataKey = "metadata"
	statusKey   = "status"
	defOffset   = 0
	defLimit    = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
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

func decodeViewUser(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewUserReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeViewProfile(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewUserReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeListUsers(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	e, err := apiutil.ReadStringQuery(r, emailKey, "")
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	s, err := apiutil.ReadStringQuery(r, statusKey, users.EnabledStatusKey)
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
	}
	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (interface{}, error) {
	req := updateUserReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}
	user.Email = strings.TrimSpace(user.Email)
	return userReq{user}, nil
}

func decodeRegisterUser(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
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

func decodeSelfRegisterUser(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return selfRegisterUserReq{user: user}, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	req.Host = r.Header.Get("Referer")
	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordChange(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := passwChangeReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeChangeUserStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeUserStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeBackup(_ context.Context, r *http.Request) (interface{}, error) {
	req := backupReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeRestore(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(apiutil.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		errors.Contains(err, users.ErrPasswordFormat),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrMissingHost,
		err == apiutil.ErrMissingPass,
		err == apiutil.ErrMissingConfPass,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidResetPass:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, errors.ErrConflict),
		errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)

	case errors.Contains(err, uuid.ErrGeneratingID),
		errors.Contains(err, users.ErrRecoveryToken):
		w.WriteHeader(http.StatusInternalServerError)

	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrRetrieveEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
