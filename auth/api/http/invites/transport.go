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
	userIDKey            = "userID"
	inviteIDKey          = "inviteID"
	inviteReponseVerbKey = "responseVerb"
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

	mux.Get("/invites/:inviteID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org_invite")(viewOrgInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/invites/:inviteID", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_org_invite")(revokeOrgInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/invites/:inviteID/:responseVerb", kithttp.NewServer(
		kitot.TraceServer(tracer, "respond_org_invite")(respondOrgInviteEndpoint(svc)),
		decodeOrgInviteResponseRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:userID/invites/received", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_invites_by_invitee")(listOrgInvitesByUserEndpoint(svc, auth.UserTypeInvitee)),
		decodeListOrgInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:userID/invites/sent", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_invites_by_inviter")(listOrgInvitesByUserEndpoint(svc, auth.UserTypeInviter)),
		decodeListOrgInvitesByUserRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/invites-platform", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_platform_invite")(createPlatformInviteEndpoint(svc)),
		decodeCreatePlatformInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites-platform", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_platform_invites")(listPlatformInvitesEndpoint(svc)),
		decodeListPlatformInvitesRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/invites-platform/:inviteID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_platform_invite")(viewPlatformInviteEndpoint(svc)),
		decodeInviteRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/invites-platform/:inviteID", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_platform_invite")(revokePlatformInviteEndpoint(svc)),
		decodeInviteRequest,
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

	if req.OrgMember.Role == "" {
		req.OrgMember.Role = auth.Viewer
	}

	if req.OrgMember.Role == auth.Owner {
		return nil, apiutil.ErrMalformedEntity
	}

	return req, nil
}

func decodeOrgInviteResponseRequest(_ context.Context, r *http.Request) (any, error) {
	req := orgInviteResponseReq{
		token:    apiutil.ExtractBearerToken(r),
		inviteID: bone.GetValue(r, inviteIDKey),
	}

	inviteResponseVerb := bone.GetValue(r, inviteReponseVerbKey)
	switch inviteResponseVerb {
	case "accept":
		req.inviteAccepted = true
	case "decline":
		req.inviteAccepted = false
	default:
		return orgInviteResponseReq{}, apiutil.ErrInvalidInviteResponse
	}

	return req, nil
}

func decodeListOrgInvitesByUserRequest(_ context.Context, r *http.Request) (any, error) {
	req := listOrgInvitesByUserReq{
		token:  apiutil.ExtractBearerToken(r),
		userID: bone.GetValue(r, userIDKey),
	}

	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

	return req, nil
}

func decodeInviteRequest(_ context.Context, r *http.Request) (any, error) {
	req := inviteReq{
		token:    apiutil.ExtractBearerToken(r),
		inviteID: bone.GetValue(r, inviteIDKey),
	}

	return req, nil
}

func decodeListOrgInvitesByOrgRequest(_ context.Context, r *http.Request) (any, error) {
	req := listOrgInvitesByOrgReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

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

	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req.pm = pm

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
		err == apiutil.ErrInvalidInviteResponse,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrInvalidQueryParams,
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
