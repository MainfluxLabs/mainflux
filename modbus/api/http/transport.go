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
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/go-kit/kit/endpoint"
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
func MakeHandler(tracer opentracing.Tracer, svc modbus.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	newServer := func(name string, e endpoint.Endpoint, decodeFunc kithttp.DecodeRequestFunc) *kithttp.Server {
		e = withIdentity(e)
		e = kitot.TraceServer(tracer, name)(e)
		return kithttp.NewServer(e, decodeFunc, encodeResponse, opts...)
	}

	r.Post("/things/:id/clients", newServer(
		"create_clients",
		createClientsEndpoint(svc),
		decodeCreateClients,
	))
	r.Get("/things/:id/clients", newServer(
		"list_clients_by_thing",
		listClientsByThingEndpoint(svc),
		decodeListClientsByThing,
	))
	r.Get("/groups/:id/clients", newServer(
		"list_clients_by_group",
		listClientsByGroupEndpoint(svc),
		decodeListClientsByGroup,
	))
	r.Get("/clients/:id", newServer(
		"view_client",
		viewClientEndpoint(svc),
		decodeRequest,
	))
	r.Put("/clients/:id", newServer(
		"update_client",
		updateClientEndpoint(svc),
		decodeUpdateClient,
	))
	r.Patch("/clients", newServer(
		"remove_clients",
		removeClientsEndpoint(svc),
		decodeRemoveClients,
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func buildPageMetadata(r *http.Request) (modbus.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	n, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")

	return modbus.PageMetadata{
		Offset: base.Offset,
		Limit:  base.Limit,
		Order:  base.Order,
		Dir:    base.Dir,
		Name:   n,
	}, nil
}

func decodeListClientsByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := buildPageMetadata(r)
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
	pm, err := buildPageMetadata(r)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
