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
	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentTypeJSON = "application/json"
	idKey           = "id"
	ctKey           = "Content-Type"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc modbus.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/things/:id/clients", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_clients")(createClientsEndpoint(svc)),
		decodeCreateClients,
		encodeResponse,
		opts...,
	))
	r.Get("/things/:id/clients", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_clients_by_thing")(listClientsByThingEndpoint(svc)),
		decodeListClientsByThing,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/clients", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_clients_by_group")(listClientsByGroupEndpoint(svc)),
		decodeListClientsByGroup,
		encodeResponse,
		opts...,
	))
	r.Get("/clients/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_client")(viewClientEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/clients/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_client")(updateClientEndpoint(svc)),
		decodeUpdateClient,
		encodeResponse,
		opts...,
	))
	r.Patch("/clients", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_clients")(removeClientsEndpoint(svc)),
		decodeRemoveClients,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("clients"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateClients(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get(ctKey), contentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createClientsReq{token: apiutil.ExtractBearerToken(r), thingID: bone.GetValue(r, idKey)}
	if err := json.NewDecoder(r.Body).Decode(&req.Clients); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListClientsByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listClientsByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, idKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeListClientsByThing(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listClientsByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, idKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeRequest(_ context.Context, r *http.Request) (any, error) {
	req := viewClientReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, idKey)}

	return req, nil
}

func decodeUpdateClient(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get(ctKey), contentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateClientReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, idKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveClients(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get(ctKey), contentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeClientsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set(ctKey, contentTypeJSON)

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
	case err == ErrMissingID,
		err == ErrInvalidScheduler,
		err == ErrMissingIPAddress,
		err == ErrMissingPort,
		err == ErrMissingDataFields,
		err == ErrInvalidFunctionCode,
		err == ErrMissingFieldName,
		err == ErrInvalidFieldType,
		err == ErrInvalidByteOrder,
		err == ErrInvalidFieldLength:
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
