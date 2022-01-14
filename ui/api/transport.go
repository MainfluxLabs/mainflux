// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/ui"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType = "text/html"
	staticDir   = "ui/web/static"
	// TODO -this is a temporary token and it will be removed once auth proxy is in place.
	token       = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NDE1MTkyNjEsImlhdCI6MTY0MTQ4MzI2MSwiaXNzIjoibWFpbmZsdXguYXV0aCIsInN1YiI6ImZscDFAZW1haWwuY29tIiwiaXNzdWVyX2lkIjoiYzkzY2FmYjMtYjNhNy00ZTdmLWE0NzAtMTVjMTRkOGVkMWUwIiwidHlwZSI6MH0.cqDOZdqiH9sXd1yuDwsv6-Mtb6_nVe_4c6cJK-iJ-Ig"
	offsetKey   = "offset"
	limitKey    = "limit"
	nameKey     = "name"
	orderKey    = "order"
	dirKey      = "dir"
	metadataKey = "metadata"
	disconnKey  = "disconnected"
	sharedKey   = "shared"
	defOffset   = 0
	defLimit    = 10
	protocol    = "http"
)

var (
	errMalformedData     = errors.New("malformed request data")
	errMalformedSubtopic = errors.New("malformed subtopic")
	redirectURL          = ""
	// channelPartRegExp    = regexp.MustCompile(`^/channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc ui.Service, redirect string, tracer opentracing.Tracer) http.Handler {
	redirectURL = redirect
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()
	r.Get("/", kithttp.NewServer(
		kitot.TraceServer(tracer, "index")(indexEndpoint(svc)),
		decodeIndexRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_things")(createThingEndpoint(svc)),
		decodeThingCreation,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_thing")(viewThingEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_thing")(updateThingEndpoint(svc)),
		decodeThingUpdate,
		encodeResponse,
		opts...,
	))

	r.Get("/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_things")(listThingsEndpoint(svc)),
		decodeListThingsRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/delete", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_thing")(removeThingEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_channels")(createChannelEndpoint(svc)),
		decodeChannelsCreation,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_channel")(viewChannelEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_channel")(updateChannelEndpoint(svc)),
		decodeChannelUpdate,
		encodeResponse,
		opts...,
	))

	r.Get("/channels", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_channels")(listChannelsEndpoint(svc)),
		decodeListChannelsRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/connect", kithttp.NewServer(
		kitot.TraceServer(tracer, "connect_thing")(connectThingEndpoint(svc)),
		decodeConnect,
		encodeResponse,
		opts...,
	))

	r.Get("/thingconn/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_connection")(listThingConnectionsEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channelconn/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_connection")(listChannelConnectionsEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/disconnect", kithttp.NewServer(
		kitot.TraceServer(tracer, "disconnect_thing")(disconnectThingEndpoint(svc)),
		decodeDisconnectThing,
		encodeResponse,
		opts...,
	))

	r.Post("/disconnect", kithttp.NewServer(
		kitot.TraceServer(tracer, "disconnect_channel")(disconnectChannelEndpoint(svc)),
		decodeDisconnectChannel,
		encodeResponse,
		opts...,
	))

	r.Post("/unassign", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign")(unassignEndpoint(svc)),
		decodeUnassignRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id/delete", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_channel")(removeChannelEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_groups")(createGroupEndpoint(svc)),
		decodeGroupCreation,
		encodeResponse,
		opts...,
	))

	r.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(listGroupsEndpoint(svc)),
		decodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group")(viewGroupEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign")(assignEndpoint(svc)),
		decodeAssignRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(updateGroupEndpoint(svc)),
		decodeGroupUpdate,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/delete", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_group")(removeGroupEndpoint(svc)),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/messages", kithttp.NewServer(
		kitot.TraceServer(tracer, "send_messages")(sendMessageEndpoint(svc)),
		decodeSendMessageRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/messages", kithttp.NewServer(
		kitot.TraceServer(tracer, "publish")(publishMessageEndpoint(svc)),
		decodePublishRequest,
		encodeResponse,
		opts...,
	))

	// r.Post("/channels/:id/messages/*", kithttp.NewServer(
	// 	kitot.TraceServer(tracer, "publish")(sendMessageEndpoint(svc)),
	// 	decodeRequest,
	// 	encodeResponse,
	// 	opts...,
	// ))

	r.GetFunc("/version", mainflux.Version("ui"))
	r.Handle("/metrics", promhttp.Handler())

	// Static file handler
	fs := http.FileServer(http.Dir(staticDir))
	r.Handle("/*", fs)

	return r
}

func decodeIndexRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	req := indexReq{
		token: getAuthorization(r),
	}

	return req, nil
}

func decodeThingCreation(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}

	req := createThingsReq{
		token:    getAuthorization(r),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}

	return req, nil
}

func getAuthorization(r *http.Request) string {
	return token
	// return r.Header.Get("Authorization")
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewResourceReq{
		token: getAuthorization(r),
		id:    bone.GetValue(r, "id"),
	}
	return req, nil
}

func decodeThingUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}

	req := updateThingReq{
		token:    getAuthorization(r),
		id:       bone.GetValue(r, "id"),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}
	return req, nil
}

func decodeListThingsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	req := listThingsReq{
		token: getAuthorization(r),
	}

	return req, nil
}

func decodeChannelsCreation(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}

	req := createChannelsReq{
		token:    getAuthorization(r),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}

	return req, nil
}

func decodeChannelUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}

	req := updateChannelReq{
		token:    getAuthorization(r),
		id:       bone.GetValue(r, "id"),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}
	return req, nil
}

func decodeListChannelsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	req := listChannelsReq{
		token: getAuthorization(r),
	}

	return req, nil
}

func decodeConnect(_ context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm()
	chanID := r.Form.Get("chanID")
	thingID := r.Form.Get("thingID")
	req := connectThingReq{
		token:   getAuthorization(r),
		ChanID:  chanID,
		ThingID: thingID,
	}
	return req, nil
}

func decodeDisconnectThing(_ context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm()
	chanID := r.Form.Get("chanID")
	thingID := r.Form.Get("thingID")
	req := disconnectThingReq{
		token:   getAuthorization(r),
		ChanID:  chanID,
		ThingID: thingID,
	}
	return req, nil
}

func decodeDisconnectChannel(_ context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm()
	chanID := r.Form.Get("chanID")
	thingID := r.Form.Get("thingID")
	req := disconnectChannelReq{
		token:   getAuthorization(r),
		ThingID: thingID,
		ChanID:  chanID,
	}
	return req, nil
}

func decodeUnassignRequest(_ context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm()
	req := unassignReq{
		assignReq{
			token:   getAuthorization(r),
			groupID: r.PostFormValue("groupId"),
			Type:    r.PostFormValue("Type"),
			Member:  r.PostFormValue("memberId"),
		},
	}
	return req, nil
}

func decodeGroupCreation(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}
	req := createGroupsReq{
		token:    getAuthorization(r),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}

	return req, nil
}

func decodeListGroupsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	req := listGroupsReq{
		token: getAuthorization(r),
	}

	return req, nil
}

func decodeAssignRequest(_ context.Context, r *http.Request) (interface{}, error) {
	memberid := r.PostFormValue("memberId")

	req := assignReq{
		token:   getAuthorization(r),
		groupID: bone.GetValue(r, "id"),
		Type:    r.PostFormValue("Type"),
		Member:  memberid,
	}
	println(req.Type)
	return req, nil
}

func decodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(r.PostFormValue("metadata")), &meta); err != nil {
		return nil, err
	}

	req := updateGroupReq{
		token:    getAuthorization(r),
		id:       bone.GetValue(r, "id"),
		Name:     r.PostFormValue("name"),
		Metadata: meta,
	}
	return req, nil
}

func decodePublishRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	chanID := r.PostFormValue("chanID")
	payload := r.PostFormValue("message")
	thingKey := r.PostFormValue("thingKey")

	msg := messaging.Message{
		Protocol: protocol,
		Channel:  chanID,
		Subtopic: "",
		Payload:  []byte(payload),
		Created:  time.Now().UnixNano(),
	}

	req := publishReq{
		msg:      msg,
		thingKey: thingKey,
		token:    getAuthorization(r),
	}

	return req, nil
}

func decodeSendMessageRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	req := sendMessageReq{
		token: getAuthorization(r),
	}

	return req, nil
}

func decodePayload(body io.ReadCloser) ([]byte, error) {
	payload, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, errMalformedData
	}
	defer body.Close()

	return payload, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)
	ar, ok := response.(uiRes)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}

	for k, v := range ar.Headers() {
		w.Header().Set(k, v)
	}
	w.WriteHeader(ar.Code())

	if ar.Empty() {
		return nil
	}
	w.Write(ar.html)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err {
	case errMalformedData, errMalformedSubtopic:
		w.WriteHeader(http.StatusBadRequest)
	case things.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	default:
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.PermissionDenied:
				w.WriteHeader(http.StatusForbidden)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}
}
