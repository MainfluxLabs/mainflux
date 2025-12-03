package invites

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

func MakeHandler(svc things.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/groups/:id/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_group_invite")(createGroupInviteEndpoint(svc)),
		decodeCreateGroupInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:id/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_group_invites_by_group")(listGroupInvitesByGroupEndpoint(svc)),
		decodeListGroupInvitesByGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_invite")(viewGroupInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/invites/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_group_invite")(revokeGroupInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/invites/:id/:action", kithttp.NewServer(
		kitot.TraceServer(tracer, "respond_group_invite")(respondGroupInviteEndpoint(svc)),
		decodeRespondGroupInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:id/invites/received", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_group_invites_by_invitee")(listGroupInvitesByUserEndpoint(svc, invites.UserTypeInvitee)),
		decodeListGroupInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:id/invites/sent", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_group_invites_by_inviter")(listGroupInvitesByUserEndpoint(svc, invites.UserTypeInviter)),
		decodeListGroupInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeCreateGroupInviteRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createGroupInviteReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRespondGroupInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := respondGroupInviteReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	action := bone.GetValue(r, invites.ResponseActionKey)
	switch action {
	case "accept":
		req.accepted = true
	case "decline":
		req.accepted = false
	default:
		return respondGroupInviteReq{}, invites.ErrInvalidInviteResponse
	}

	return req, nil
}

func decodeListGroupInvitesByUserRequest(_ context.Context, r *http.Request) (any, error) {
	req := listGroupInvitesByUserReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	pm, err := invites.BuildPageMetadataInvites(r)
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

func decodeListGroupInvitesByGroupRequest(_ context.Context, r *http.Request) (any, error) {
	req := listGroupInvitesByGroupReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	pm, err := invites.BuildPageMetadataInvites(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

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
	case errors.Contains(err, invites.ErrInvalidInviteResponse):
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
