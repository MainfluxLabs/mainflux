package members

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
	orgIDKey  = "orgID"
	memberKey = "memberID"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Get("/orgs/:orgID/members/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_member")(viewMemberEndpoint(svc)),
		decodeMemberRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_members")(assignMembersEndpoint(svc)),
		decodeMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/orgs/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_members")(unassignMembersEndpoint(svc)),
		decodeUnassignMembers,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_members")(updateMembersEndpoint(svc)),
		decodeMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_members_by_org")(listMembersByOrgEndpoint(svc)),
		decodeListMembersByOrg,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeListMembersByOrg(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	req := listMembersByOrgReq{
		token:  apiutil.ExtractBearerToken(r),
		id:     bone.GetValue(r, apiutil.IDKey),
		offset: o,
		limit:  l,
	}
	return req, nil
}

func decodeMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := membersReq{
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

func decodeMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := memberReq{
		token:    apiutil.ExtractBearerToken(r),
		orgID:    bone.GetValue(r, orgIDKey),
		memberID: bone.GetValue(r, memberKey),
	}

	return req, nil
}

func decodeUnassignMembers(_ context.Context, r *http.Request) (interface{}, error) {
	req := unassignMembersReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

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
		err == apiutil.ErrMissingMemberID,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrInvalidRole,
		err == apiutil.ErrInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, auth.ErrOrgMemberAlreadyAssigned):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
