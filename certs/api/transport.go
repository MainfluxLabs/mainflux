// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, tracer opentracing.Tracer, pkiAgent pki.Agent, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/certs", kithttp.NewServer(
		kitot.TraceServer(tracer, "issue_cert")(issueCertEndpoint(svc)),
		decodeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/certs/:serial", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_cert")(viewCertEndpoint(svc)),
		decodeViewCert,
		encodeResponse,
		opts...,
	))

	r.Delete("/certs/:serial", kithttp.NewServer(
		kitot.TraceServer(tracer, "revoke_cert")(revokeCertEndpoint(svc)),
		decodeRevokeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/serials", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_serials")(listSerialsByThingEndpoint(svc)),
		decodeListSerialsByThing,
		encodeResponse,
		opts...,
	))

	r.Put("/certs/:serial", kithttp.NewServer(
		kitot.TraceServer(tracer, "renew_cert")(renewCertEndpoint(svc)),
		decodeViewCert,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("certs"))

	return r
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

func decodeListSerialsByThing(_ context.Context, r *http.Request) (any, error) {
	l, err := apiutil.ReadUintQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}
	o, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeViewCert(_ context.Context, r *http.Request) (any, error) {
	req := viewReq{
		token:  apiutil.ExtractBearerToken(r),
		serial: bone.GetValue(r, apiutil.SerialKey),
	}

	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (any, error) {
	if r.Header.Get("Content-Type") != apiutil.ContentTypeJSON {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := addCertsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeRevokeCerts(_ context.Context, r *http.Request) (any, error) {
	req := revokeReq{
		token:  apiutil.ExtractBearerToken(r),
		serial: bone.GetValue(r, apiutil.SerialKey),
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	apiutil.EncodeError(err, w)
	apiutil.WriteErrorResponse(err, w)
}
