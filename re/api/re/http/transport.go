//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/re"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc re.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Get("/", kithttp.NewServer(
		kitot.TraceServer(tracer, "info")(infoEndpoint(svc)),
		decodeGet,
		encodeResponse,
		opts...,
	))

	r.Put("/streams/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "view")(updateEndpoint(svc)),
		decodeUpdateStream,
		encodeResponse,
		opts...,
	))

	r.Post("/streams", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_stream")(createEndpoint(svc)),
		decodeCreateStream,
		encodeResponse,
		opts...,
	))

	r.Get("/streams", kithttp.NewServer(
		kitot.TraceServer(tracer, "list")(listEndpoint(svc)),
		decodeGet,
		encodeResponse,
		opts...,
	))

	r.Get("/streams/:name", kithttp.NewServer(
		kitot.TraceServer(tracer, "view")(viewEndpoint(svc)),
		decodeViewStream,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("things"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeGet(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeViewStream(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewStreamReq{
		// token: r.Header.Get("Authorization"),
		name: bone.GetValue(r, "name"),
	}
	return req, nil
}

func decodeCreateStream(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := streamReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeUpdateStream(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errUnsupportedContentType
	}

	req := streamReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	req.Name = bone.GetValue(r, "name")

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
	w.Header().Set("Content-Type", contentType)

	switch err {
	case re.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case re.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case io.ErrUnexpectedEOF:
		w.WriteHeader(http.StatusBadRequest)
	case io.EOF:
		w.WriteHeader(http.StatusBadRequest)
	default:
		switch err.(type) {
		case *json.SyntaxError:
			w.WriteHeader(http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func readUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errInvalidQueryParams
	}

	return val, nil
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidQueryParams
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
}
