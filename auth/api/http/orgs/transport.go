package orgs

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
	contentType = "application/json"
	maxNameSize = 254
	offsetKey   = "offset"
	limitKey    = "limit"
	metadataKey = "metadata"
	nameKey     = "name"
	defOffset   = 0
	defLimit    = 10
	orgIDKey    = "orgID"
	memberKey   = "memberID"
	idKey       = "id"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	mux.Post("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_orgs")(createOrgsEndpoint(svc)),
		decodeCreateOrgs,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org")(viewOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_org")(updateOrgEndpoint(svc)),
		decodeUpdateOrg,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_org")(deleteOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_orgs")(listOrgsEndpoint(svc)),
		decodeListOrgs,
		encodeResponse,
		opts...,
	))

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

	mux.Get("/members/:id/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_orgs_by_member")(listOrgsByMemberEndpoint(svc)),
		decodeListOrgsByMember,
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

	return mux
}

func decodeListOrgs(_ context.Context, r *http.Request) (interface{}, error) {
	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	req := listOrgsReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, orgIDKey),
		name:     n,
		offset:   o,
		limit:    l,
		metadata: m,
	}

	return req, nil
}

func decodeListMembersByOrg(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	req := listMembersByOrgReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, idKey),
		offset:   o,
		limit:    l,
		metadata: m,
	}
	return req, nil
}

func decodeListOrgsByMember(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	req := listOrgsByMemberReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, idKey),
		name:     n,
		offset:   o,
		limit:    l,
		metadata: m,
	}

	return req, nil
}

func decodeCreateOrgs(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createOrgsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateOrg(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateOrgReq{
		id:    bone.GetValue(r, idKey),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeOrgRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := orgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}

	return req, nil
}

func decodeMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := membersReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, idKey),
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
		orgID: bone.GetValue(r, idKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeBackup(_ context.Context, r *http.Request) (interface{}, error) {
	req := backupReq{
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeRestore(_ context.Context, r *http.Request) (interface{}, error) {
	req := restoreReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

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
		err == apiutil.ErrMissingID,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrInvalidMemberRole,
		err == apiutil.ErrInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, auth.ErrOrgMemberAlreadyAssigned):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)

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
