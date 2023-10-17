package orgs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/logger"
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
	adminKey    = "admin"
	defOffset   = 0
	defLimit    = 10
	orgIDKey    = "orgID"
	memberKey   = "memberID"
	groupIDKey  = "groupID"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	mux.Post("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_org")(createOrgEndpoint(svc)),
		decodeOrgCreate,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org")(viewOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:orgID", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_org")(updateOrgEndpoint(svc)),
		decodeOrgUpdate,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:orgID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_org")(deleteOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_orgs")(listOrgsEndpoint(svc)),
		decodeListOrgsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID/members/:memberID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_member")(viewMemberEndpoint(svc)),
		decodeMemberRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:orgID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_members")(assignMembersEndpoint(svc)),
		decodeMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:orgID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_members")(unassignMembersEndpoint(svc)),
		decodeUnassignMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:orgID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_members")(updateMembersEndpoint(svc)),
		decodeMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_members")(listMembersEndpoint(svc)),
		decodeListMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/orgs/:orgID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_groups")(assignOrgGroupsEndpoint(svc)),
		decodeGroupsRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:orgID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_groups")(unassignOrgGroupsEndpoint(svc)),
		decodeGroupsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:orgID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(listGroupsEndpoint(svc)),
		decodeListGroupsRequest,
		encodeResponse,
		opts...,
	))
	mux.Post("/orgs/:orgID/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_policies")(createPoliciesEndpint(svc)),
		decodeCreatePoliciesRequest,
		encodeResponse,
		opts...,
	))
	mux.Get("/members/:memberID/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_memberships")(listMemberships(svc)),
		decodeListMembershipsRequest,
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

func decodeListOrgsRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	a, err := apiutil.ReadBoolQuery(r, adminKey, false)
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
		admin:    a,
	}

	return req, nil
}

func decodeListMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	req := listOrgMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, orgIDKey),
		offset:   o,
		limit:    l,
		metadata: m,
	}
	return req, nil
}

func decodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	req := listOrgGroupsReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, orgIDKey),
		offset:   o,
		limit:    l,
		metadata: m,
	}
	return req, nil
}

func decodeCreatePoliciesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createPoliciesReq{
		token:   apiutil.ExtractBearerToken(r),
		orgID:   bone.GetValue(r, orgIDKey),
		groupID: bone.GetValue(r, groupIDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListMembershipsRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	req := listOrgMembershipsReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, "memberID"),
		name:     n,
		offset:   o,
		limit:    l,
		metadata: m,
	}

	return req, nil
}

func decodeOrgCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createOrgReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeOrgUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateOrgReq{
		id:    bone.GetValue(r, orgIDKey),
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
		id:    bone.GetValue(r, orgIDKey),
	}

	return req, nil
}

func decodeMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := membersReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, orgIDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	for i := range req.Members {
		if req.Members[i].Role == "" {
			req.Members[i].Role = auth.ViewerRole
		}

		if req.Members[i].Role == auth.OwnerRole {
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

func decodeUnassignMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := unassignMembersReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, orgIDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := groupsReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, orgIDKey),
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

	if ar, ok := response.(mainflux.Response); ok {
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
