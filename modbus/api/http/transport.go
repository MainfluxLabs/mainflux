// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
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
	idKey        = "id"
	ctKey        = "Content-Type"
	ipAddressKey = "ip_address"
	portKey      = "port"
	slaveIDKey   = "slave_id"
	frequencyKey = "frequency"
	clientKey    = "client"
	fileKey      = "file"
	maxMemory    = 32 << 20
	csvExt       = ".csv"
	jsonExt      = ".json"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc modbus.Service, ac domain.AuthClient, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	withIdentity := authn.IdentityMiddleware(ac, logger)

	r.Post("/things/:id/clients", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "create_clients"),
			withIdentity,
		)(createClientsEndpoint(svc)),
		decodeCreateClients,
		encodeResponse,
		opts...,
	))
	r.Get("/things/:id/clients", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_clients_by_thing"),
			withIdentity,
		)(listClientsByThingEndpoint(svc)),
		decodeListClientsByThing,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/clients", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_clients_by_group"),
			withIdentity,
		)(listClientsByGroupEndpoint(svc)),
		decodeListClientsByGroup,
		encodeResponse,
		opts...,
	))
	r.Get("/clients/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_client"),
			withIdentity,
		)(viewClientEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))
	r.Put("/clients/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "update_client"),
			withIdentity,
		)(updateClientEndpoint(svc)),
		decodeUpdateClient,
		encodeResponse,
		opts...,
	))
	r.Patch("/clients", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_clients"),
			withIdentity,
		)(removeClientsEndpoint(svc)),
		decodeRemoveClients,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("clients"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

// decodeCreateClients dispatches on Content-Type so JSON bulk-create and file import share one route.
func decodeCreateClients(ctx context.Context, r *http.Request) (any, error) {
	ct := r.Header.Get(ctKey)
	switch {
	case strings.Contains(ct, apiutil.ContentTypeJSON):
		req := createClientsReq{token: apiutil.ExtractBearerToken(r), thingID: bone.GetValue(r, idKey)}
		if err := json.NewDecoder(r.Body).Decode(&req.Clients); err != nil {
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}

		return req, nil
	case strings.Contains(ct, apiutil.ContentTypeMultipart):
		return decodeClientFile(ctx, r)
	default:
		return nil, apiutil.ErrUnsupportedContentType
	}
}

// decodeClientFile builds one client from its config part plus data fields parsed from the file part.
func decodeClientFile(_ context.Context, r *http.Request) (any, error) {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	var c client
	if err := json.Unmarshal([]byte(r.FormValue(clientKey)), &c); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	file, header, err := r.FormFile(fileKey)
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}
	data = bytes.TrimPrefix(data, apiutil.UTF8BOM)

	var fields []field
	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case csvExt:
		fields, err = parseCSVFields(bytes.NewReader(data))
	case jsonExt:
		err = json.Unmarshal(data, &fields)
	default:
		return nil, ErrUnsupportedFileFormat
	}

	if err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	c.DataFields = normalizeFieldCase(fields)

	req := createClientsReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, idKey),
		Clients: []client{c},
	}

	return req, nil
}

// normalizeFieldCase normalizes fields in imported files to avoid case sensitivity.
func normalizeFieldCase(fields []field) []field {
	for i := range fields {
		fields[i].Type = strings.ToLower(strings.TrimSpace(fields[i].Type))
		fields[i].ByteOrder = strings.ToUpper(strings.TrimSpace(fields[i].ByteOrder))
	}
	return fields
}

func buildPageMetadata(r *http.Request) (modbus.PageMetadata, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	n, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	ip, err := apiutil.ReadStringQuery(r, ipAddressKey, "")
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	p, err := apiutil.ReadStringQuery(r, portKey, "")
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	slaveID, err := apiutil.ReadIntQuery(r, slaveIDKey, -1)
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	f, err := apiutil.ReadStringQuery(r, frequencyKey, "")
	if err != nil {
		return modbus.PageMetadata{}, err
	}

	return modbus.PageMetadata{
		Offset:    base.Offset,
		Limit:     base.Limit,
		Order:     base.Order,
		Dir:       base.Dir,
		Name:      n,
		IPAddress: ip,
		Port:      p,
		SlaveID:   slaveID,
		Frequency: f,
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
	if !strings.Contains(r.Header.Get(ctKey), apiutil.ContentTypeJSON) {
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
	if !strings.Contains(r.Header.Get(ctKey), apiutil.ContentTypeJSON) {
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
	w.Header().Set(ctKey, apiutil.ContentTypeJSON)

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
		err == ErrInvalidFieldLength,
		err == ErrUnsupportedFileFormat:
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
