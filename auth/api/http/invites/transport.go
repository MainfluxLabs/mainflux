package invites

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

const (
	responseActionKey = "action"
	stateKey          = "state"
)

func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/orgs/:id/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_org_invite")(createOrgInviteEndpoint(svc)),
		decodeCreateOrgInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_invites_by_org")(listOrgInvitesByOrgEndpoint(svc)),
		decodeListOrgInvitesByOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org_invite")(viewOrgInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_org_invite")(revokeOrgInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/invites/:id/:action", kithttp.NewServer(
		kitot.TraceServer(tracer, "respond_org_invite")(respondOrgInviteEndpoint(svc)),
		decodeRespondOrgInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:id/invites/received", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_invites_by_invitee")(listOrgInvitesByUserEndpoint(svc, auth.UserTypeInvitee)),
		decodeListOrgInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:id/invites/sent", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_invites_by_inviter")(listOrgInvitesByUserEndpoint(svc, auth.UserTypeInviter)),
		decodeListOrgInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeCreateOrgInviteRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createOrgInviteReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRespondOrgInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := respondOrgInviteReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	action := bone.GetValue(r, responseActionKey)
	switch action {
	case "accept":
		req.accepted = true
	case "decline":
		req.accepted = false
	default:
		return respondOrgInviteReq{}, auth.ErrInvalidInviteResponse
	}

	return req, nil
}

func decodeListOrgInvitesByUserRequest(_ context.Context, r *http.Request) (any, error) {
	req := listOrgInvitesByUserReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	pm, err := buildPageMetadataInvites(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

	return req, nil
}

func decodeInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := inviteReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeListOrgInvitesByOrgRequest(_ context.Context, r *http.Request) (any, error) {
	req := listOrgInvitesByOrgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	pm, err := buildPageMetadataInvites(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

	return req, nil
}

func buildPageMetadataInvites(r *http.Request) (auth.PageMetadataInvites, error) {
	pm := auth.PageMetadataInvites{}

	apm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return auth.PageMetadataInvites{}, err
	}

	pm.PageMetadata = apm

	state, err := apiutil.ReadStringQuery(r, stateKey, "")
	if err != nil {
		return auth.PageMetadataInvites{}, err
	}

	pm.State = state

	return pm, nil
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
	case errors.Contains(err, auth.ErrInvalidInviteResponse):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, auth.ErrOrgMembershipExists):
		w.WriteHeader(http.StatusConflict)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
