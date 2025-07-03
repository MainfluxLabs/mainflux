package memberships

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
	emailKey  = "email"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux.Post("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_memberships")(createMembershipsEndpoint(svc)),
		decodeMembershipsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID/members/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_membership")(viewMembershipEndpoint(svc)),
		decodeMembershipRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_memberships_by_org")(listMembershipsByOrgEndpoint(svc)),
		decodeListMemberships,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_memberships")(updateMembershipsEndpoint(svc)),
		decodeMembershipsRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_memberships")(removeMembershipsEndpoint(svc)),
		decodeRemoveMemberships,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeListMemberships(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	e, err := apiutil.ReadStringQuery(r, emailKey, "")
	or, err := apiutil.ReadStringQuery(r, apiutil.OrderKey, apiutil.IDOrder)
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, apiutil.DirKey, apiutil.DescDir)
	if err != nil {
		return nil, err
	}

	req := listMembershipsReq{
		token:  apiutil.ExtractBearerToken(r),
		id:     bone.GetValue(r, apiutil.IDKey),
		email:  e,
		offset: o,
		limit:  l,
		order:  or,
		dir:    d,
	}
	return req, nil
}

func decodeMembershipsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := membershipsReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	for i := range req.OrgMemberships {
		if req.OrgMemberships[i].Role == "" {
			req.OrgMemberships[i].Role = auth.Viewer
		}

		if req.OrgMemberships[i].Role == auth.Owner {
			return nil, apiutil.ErrMalformedEntity
		}
	}

	return req, nil
}

func decodeMembershipRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := membershipReq{
		token:    apiutil.ExtractBearerToken(r),
		orgID:    bone.GetValue(r, orgIDKey),
		memberID: bone.GetValue(r, memberKey),
	}

	return req, nil
}

func decodeRemoveMemberships(_ context.Context, r *http.Request) (interface{}, error) {
	req := removeMembershipsReq{
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
	case errors.Contains(err, auth.ErrMembershipExists):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
