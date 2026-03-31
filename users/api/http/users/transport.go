// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

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

const (
	emailKey        = "email"
	statusKey       = "status"
	emailTokenKey   = "token"
	stateKey        = "state"
	providerKey     = "provider"
	codeKey         = "code"
	verifierKey     = "verifier"
	inviteIDKey     = "invite_id"
	redirectPathKey = "redirect_path"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger, passwordRegex *regexp.Regexp) *bone.Mux {
	userPasswordRegex = passwordRegex

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	callbackOpts := append([]kithttp.ServerOption{}, opts...)
	callbackOpts = append(callbackOpts, kithttp.ServerErrorEncoder(encodeOAuthCallbackError))

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

	mux.Get("/users/oauth/:provider", kithttp.NewServer(
		kitot.TraceServer(tracer, "oauth_login")(oauthLoginEndpoint(svc)),
		decodeOAuthLogin,
		encodeOAuthLoginResponse,
		opts...,
	))

	mux.Get("/users/oauth/:provider/callback", kithttp.NewServer(
		kitot.TraceServer(tracer, "oauth_callback")(oauthCallbackEndpoint(svc)),
		decodeOAuthCallback,
		encodeOAuthCallbackResponse,
		callbackOpts...,
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
	base, err := apiutil.BuildPageMetadata(r)
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

	pm := users.PageMetadata{
		Offset:   base.Offset,
		Limit:    base.Limit,
		Order:    base.Order,
		Dir:      base.Dir,
		Email:    e,
		Status:   s,
		Metadata: m,
	}

	req := listUsersReq{
		token: apiutil.ExtractBearerToken(r),
		pm:    pm,
	}
	return req, nil
}

func decodeSearchUsers(_ context.Context, r *http.Request) (any, error) {
	pm := users.PageMetadata{
		Offset: apiutil.DefOffset,
		Limit:  apiutil.DefLimit,
		Order:  apiutil.IDOrder,
		Dir:    apiutil.DescDir,
		Status: users.EnabledStatusKey,
	}

	req := listUsersReq{
		token: apiutil.ExtractBearerToken(r),
		pm:    pm,
	}

	if r.Body == nil || r.ContentLength == 0 {
		return req, nil
	}

	var upm users.PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&upm); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	if upm.Offset > 0 {
		req.pm.Offset = upm.Offset
	}

	if upm.Limit > 0 {
		req.pm.Limit = upm.Limit
	}

	if upm.Order != "" {
		req.pm.Order = upm.Order
	}

	if upm.Dir != "" {
		req.pm.Dir = upm.Dir
	}

	if upm.Status != "" {
		req.pm.Status = upm.Status
	}

	if upm.Email != "" {
		req.pm.Email = upm.Email
	}

	if upm.Metadata != nil {
		req.pm.Metadata = upm.Metadata
	}

	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (any, error) {
	req := updateUserReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeOAuthLogin(_ context.Context, r *http.Request) (any, error) {
	req := oauthLoginReq{
		provider:     bone.GetValue(r, providerKey),
		inviteID:     r.URL.Query().Get(inviteIDKey),
		redirectPath: r.URL.Query().Get(redirectPathKey),
	}

	return req, nil
}

func decodeOAuthCallback(_ context.Context, r *http.Request) (any, error) {
	stateCookie, err := r.Cookie(stateKey)
	if err != nil {
		return nil, apiutil.ErrInvalidState
	}

	verifierCookie, err := r.Cookie(verifierKey)
	if err != nil {
		return nil, apiutil.ErrInvalidState
	}

	var inviteID, redirectPath string
	if inviteCookie, err := r.Cookie(inviteIDKey); err == nil {
		inviteID = inviteCookie.Value
	}
	if redirectCookie, err := r.Cookie(redirectPathKey); err == nil {
		redirectPath = redirectCookie.Value
	}

	req := oauthCallbackReq{
		provider:      bone.GetValue(r, providerKey),
		code:          r.URL.Query().Get(codeKey),
		state:         r.URL.Query().Get(stateKey),
		originalState: stateCookie.Value,
		verifier:      verifierCookie.Value,
		inviteID:      inviteID,
		redirectPath:  redirectPath,
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

func decodePasswordResetRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodePasswordChange(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := passwChangeReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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

func encodeOAuthLoginResponse(_ context.Context, w http.ResponseWriter, response any) error {
	res := response.(oauthLoginRes)

	http.SetCookie(w, &http.Cookie{
		Name:     stateKey,
		Value:    res.State,
		Path:     "/users/oauth/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     verifierKey,
		Value:    res.Verifier,
		Path:     "/users/oauth/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	if res.InviteID != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     inviteIDKey,
			Value:    res.InviteID,
			Path:     "/users/oauth/",
			MaxAge:   300,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     redirectPathKey,
			Value:    res.RedirectPath,
			Path:     "/users/oauth/",
			MaxAge:   300,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	return json.NewEncoder(w).Encode(redirectURLRes{RedirectURL: res.RedirectURL})
}

func encodeOAuthCallbackResponse(_ context.Context, w http.ResponseWriter, response any) error {
	for _, name := range []string{stateKey, verifierKey, inviteIDKey, redirectPathKey} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Path:     "/users/oauth/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	res := response.(redirectURLRes)
	w.Header().Set("Location", res.RedirectURL)
	w.WriteHeader(http.StatusFound)
	return nil
}

func encodeOAuthCallbackError(ctx context.Context, err error, w http.ResponseWriter) {
	for _, name := range []string{stateKey, verifierKey, inviteIDKey, redirectPathKey} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Path:     "/users/oauth/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	encodeError(ctx, err, w)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, errors.ErrPasswordFormat),
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
