package memberships

import (
	"context"
	"encoding/json"
	"io"
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
		kitot.TraceServer(tracer, "create_org_memberships")(createOrgMembershipsEndpoint(svc)),
		decodeOrgMembershipsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/memberships/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_org_memberships")(backupOrgMembershipsEndpoint(svc)),
		decodeBackupByOrg,
		apiutil.EncodeFileResponse,
		opts...,
	))

	mux.Post("/orgs/:id/memberships/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_org_memberships")(restoreOrgMembershipsEndpoint(svc)),
		decodeRestoreByOrg,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID/members/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org_membership")(viewOrgMembershipEndpoint(svc)),
		decodeOrgMembershipRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_org_memberships")(listOrgMembershipsEndpoint(svc)),
		decodeListOrgMemberships,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_org_memberships")(updateOrgMembershipsEndpoint(svc)),
		decodeOrgMembershipsRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/orgs/:id/memberships", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_org_memberships")(removeOrgMembershipsEndpoint(svc)),
		decodeRemoveOrgMemberships,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeListOrgMemberships(_ context.Context, r *http.Request) (any, error) {
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

	or, err := apiutil.ReadStringQuery(r, apiutil.OrderKey, apiutil.IDOrder)
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, apiutil.DirKey, apiutil.DescDir)
	if err != nil {
		return nil, err
	}

	req := listOrgMembershipsReq{
		token:  apiutil.ExtractBearerToken(r),
		orgID:  bone.GetValue(r, apiutil.IDKey),
		email:  e,
		offset: o,
		limit:  l,
		order:  or,
		dir:    d,
	}
	return req, nil
}

func decodeOrgMembershipsRequest(_ context.Context, r *http.Request) (any, error) {
	req := orgMembershipsReq{
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

func decodeOrgMembershipRequest(_ context.Context, r *http.Request) (any, error) {
	req := orgMembershipReq{
		token:    apiutil.ExtractBearerToken(r),
		orgID:    bone.GetValue(r, orgIDKey),
		memberID: bone.GetValue(r, memberKey),
	}

	return req, nil
}

func decodeRemoveOrgMemberships(_ context.Context, r *http.Request) (any, error) {
	req := removeOrgMembershipsReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeBackupByOrg(_ context.Context, r *http.Request) (any, error) {
	req := backupByOrgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	return req, nil
}

func decodeRestoreByOrg(ctx context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeOctetStream) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreByOrgReq{
		id:    bone.GetValue(r, apiutil.IDKey),
		token: apiutil.ExtractBearerToken(r),
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	if err := json.Unmarshal(data, &req.OrgMemberships); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

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
	case errors.Contains(err, auth.ErrOrgMembershipExists):
		w.WriteHeader(http.StatusConflict)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
