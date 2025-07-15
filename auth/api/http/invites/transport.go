package invites

import (
	"context"
	"encoding/json"
	"net/http"

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
	orgIDKey             = "orgID"
	userIDKey            = "userID"
	inviteIDKey          = "inviteID"
	inviteReponseVerbKey = "responseVerb"
)

func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/orgs/:id/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "invite_members")(inviteMembersEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:orgID/invites/:inviteID", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_invite")(revokeInviteEndpoint(svc)),
		decodeInviteRevokeRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:orgID/invites/:inviteID/:responseVerb", kithttp.NewServer(
		kitot.TraceServer(tracer, "respond_invite")(respondInviteEndpoint(svc)),
		decodeInviteResponseRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:userID/invites", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_invites_by_user")(listInvitesByUserEndpoint(svc)),
		decodeListInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := invitesReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	for i := range req.OrgMembers {
		if req.OrgMembers[i].Role == "" {
			req.OrgMembers[i].Role = auth.Viewer
		}

		if req.OrgMembers[i].Role == auth.Owner {
			return nil, apiutil.ErrMalformedEntity
		}
	}

	return req, nil
}

func decodeInviteRevokeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := inviteRevokeReq{
		token:    apiutil.ExtractBearerToken(r),
		orgID:    bone.GetValue(r, orgIDKey),
		inviteID: bone.GetValue(r, inviteIDKey),
	}

	return req, nil
}

func decodeInviteResponseRequest(_ context.Context, r *http.Request) (any, error) {
	req := inviteResponseReq{
		token:    apiutil.ExtractBearerToken(r),
		orgID:    bone.GetValue(r, orgIDKey),
		inviteID: bone.GetValue(r, inviteIDKey),
	}

	inviteResponseVerb := bone.GetValue(r, inviteReponseVerbKey)
	switch inviteResponseVerb {
	case "accept":
		req.inviteAccepted = true
	case "decline":
		req.inviteAccepted = false
	default:
		return inviteResponseReq{}, apiutil.ErrInvalidInviteResponse
	}

	return req, nil
}

func decodeListInvitesByUserRequest(_ context.Context, r *http.Request) (any, error) {
	req := listInvitesByUserReq{
		token:  apiutil.ExtractBearerToken(r),
		userID: bone.GetValue(r, userIDKey),
	}

	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadUintQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	req.offset = offset
	req.limit = limit

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

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingOrgID,
		err == apiutil.ErrMissingInviteID,
		err == apiutil.ErrMissingMemberID,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrInvalidRole:
		w.WriteHeader(http.StatusBadRequest)
	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, auth.ErrOrgMembershipExists),
		errors.Contains(err, auth.ErrUserAlreadyInvited):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
