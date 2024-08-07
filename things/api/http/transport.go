// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	nameKey     = "name"
	orderKey    = "order"
	dirKey      = "dir"
	metadataKey = "metadata"
	orgKey      = "org_id"
	idKey       = "id"
	defOffset   = 0
	defLimit    = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc things.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/groups/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_things")(createThingsEndpoint(svc)),
		decodeCreateThings,
		encodeResponse,
		opts...,
	))

	r.Patch("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_things")(removeThingsEndpoint(svc)),
		decodeRemoveThings,
		encodeResponse,
		opts...,
	))

	r.Patch("/things/:id/key", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_key")(updateKeyEndpoint(svc)),
		decodeUpdateKey,
		encodeResponse,
		opts...,
	))

	r.Put("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_thing")(updateThingEndpoint(svc)),
		decodeUpdateThing,
		encodeResponse,
		opts...,
	))

	r.Delete("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_thing")(removeThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_thing")(viewThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_channel_by_thing")(viewChannelByThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things")(listThingsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Post("/things/search", kithttp.NewServer(
		kitot.TraceServer(tracer, "search_things")(listThingsEndpoint(svc)),
		decodeListByMetadata,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_channels")(createChannelsEndpoint(svc)),
		decodeCreateChannels,
		encodeResponse,
		opts...,
	))

	r.Patch("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_channels")(removeChannelsEndpoint(svc)),
		decodeRemoveChannels,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_channel")(updateChannelEndpoint(svc)),
		decodeUpdateChannel,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_channel")(removeChannelEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_channel")(viewChannelEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_channel")(listThingsByChannelEndpoint(svc)),
		decodeListByConnection,
		encodeResponse,
		opts...,
	))

	r.Get("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_channels")(listChannelsEndpoint(svc)),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Post("/connect", kithttp.NewServer(
		kitot.TraceServer(tracer, "connect")(connectEndpoint(svc)),
		decodeConnectionsList,
		encodeResponse,
		opts...,
	))

	r.Put("/disconnect", kithttp.NewServer(
		kitot.TraceServer(tracer, "disconnect")(disconnectEndpoint(svc)),
		decodeConnectionsList,
		encodeResponse,
		opts...,
	))

	r.Post("/orgs/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_groups")(createGroupsEndpoint(svc)),
		decodeCreateGroups,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group")(viewGroupEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Put("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(updateGroupEndpoint(svc)),
		decodeUpdateGroup,
		encodeResponse,
		opts...,
	))

	r.Delete("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_group")(removeGroupEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(listGroupsEndpoint(svc)),
		decodeListGroups,
		encodeResponse,
		opts...,
	))

	r.Patch("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_groups")(removeGroupsEndpoint(svc)),
		decodeRemoveGroups,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things_by_group")(listThingsByGroupEndpoint(svc)),
		decodeListMembers,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_by_thing")(viewGroupByThingEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_channels_by_group")(listChannelsByGroupEndpoint(svc)),
		decodeListMembers,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group_by_channel")(viewGroupByChannelEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_roles_by_group")(createRolesByGroupEndpoint(svc)),
		decodeGroupRoles,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_roles_by_group")(listRolesByGroupEndpoint(svc)),
		decodeListGroupRoles,
		encodeResponse,
		opts...,
	))

	r.Put("/groups/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_roles_by_group")(updateRolesByGroupEndpoint(svc)),
		decodeGroupRoles,
		encodeResponse,
		opts...,
	))

	r.Patch("/groups/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_roles_by_group")(removeRolesByGroupEndpoint(svc)),
		decodeRemoveGroupRoles,
		encodeResponse,
		opts...,
	))

	r.Get("/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup")(backupEndpoint(svc)),
		decodeBackup,
		encodeResponse,
		opts...,
	))

	r.Post("/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore")(restoreEndpoint(svc)),
		decodeRestore,
		encodeResponse,
		opts...,
	))

	r.Post("/identify", kithttp.NewServer(
		kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
		decodeIdentify,
		encodeResponse,
		opts...,
	))

	r.Post("/connections", kithttp.NewServer(
		kitot.TraceServer(tracer, "get_conn_by_key")(getConnByKeyEndpoint(svc)),
		decodeGetConnByKey,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("things"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateThings(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createThingsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Things); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateThing(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateThingReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateKey(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateKeyReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCreateChannels(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createChannelsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Channels); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateChannel(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateChannelReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveChannels(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeChannelsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := resourceReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	n, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	or, err := apiutil.ReadStringQuery(r, orderKey, "")
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, dirKey, "")
	if err != nil {
		return nil, err
	}

	m, err := apiutil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	req := listResourcesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMetadata: things.PageMetadata{
			Offset:   o,
			Limit:    l,
			Name:     n,
			Order:    or,
			Dir:      d,
			Metadata: m,
		},
	}

	return req, nil
}

func decodeListByMetadata(_ context.Context, r *http.Request) (interface{}, error) {
	req := listResourcesReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req.pageMetadata); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListByConnection(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadLimitQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	or, err := apiutil.ReadStringQuery(r, orderKey, "")
	if err != nil {
		return nil, err
	}

	d, err := apiutil.ReadStringQuery(r, dirKey, "")
	if err != nil {
		return nil, err
	}

	req := listByConnectionReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
		pageMetadata: things.PageMetadata{
			Offset: o,
			Limit:  l,
			Order:  or,
			Dir:    d,
		},
	}

	return req, nil
}

func decodeConnectionsList(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := connectionsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListMembers(_ context.Context, r *http.Request) (interface{}, error) {
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

	req := listMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		id:       bone.GetValue(r, idKey),
		offset:   o,
		limit:    l,
		metadata: m,
	}

	return req, nil
}

func decodeCreateGroups(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		orgID: bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Groups); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListGroups(_ context.Context, r *http.Request) (interface{}, error) {
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

	orgID, err := apiutil.ReadStringQuery(r, orgKey, "")
	if err != nil {
		return nil, err
	}

	req := listGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		pageMetadata: things.PageMetadata{
			Offset:   o,
			Limit:    l,
			Metadata: m,
			Name:     n,
		},
		orgID: orgID,
	}
	return req, nil
}

func decodeUpdateGroup(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateGroupReq{
		id:    bone.GetValue(r, idKey),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveGroups(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeGroupsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveThings(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeThingsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeBackup(_ context.Context, r *http.Request) (interface{}, error) {
	req := backupReq{token: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeRestore(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := restoreReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeIdentify(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := identifyReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeGetConnByKey(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := getConnByKeyReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeGroupRoles(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := groupRolesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListGroupRoles(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	req := listGroupMembersReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
		offset:  o,
		limit:   l,
	}

	return req, nil
}

func decodeRemoveGroupRoles(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeGroupRolesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, idKey),
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
	// ErrNotFound can be masked by ErrAuthentication, but it has priority.
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken,
		err == apiutil.ErrBearerKey:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrNameSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrInvalidIDFormat:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)

	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrRetrieveEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)

	case errors.Contains(err, uuid.ErrGeneratingID):
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
