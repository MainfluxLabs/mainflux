// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	adapter "github.com/MainfluxLabs/mainflux/converters"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	maxMemory            = 32 << 20
	fileKey              = "file"
	multiPartContentType = "multipart/form-data"
)

var utf8BOM = []byte{0xef, 0xbb, 0xbf}

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, ac domain.AuthClient, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	r := bone.New()

	r.Post("/csv", kithttp.NewServer(
		kitot.TraceServer(tracer, "convert_csv")(convertCSVEndpoint(svc)),
		decodeConvertCSVFile,
		encodeResponse,
		opts...,
	))

	r.Post("/json", kithttp.NewServer(
		kitot.TraceServer(tracer, "convert_json")(convertJSONEndpoint(svc)),
		decodeConvertJSONFile,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("converters"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeConvertCSVFile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), multiPartContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, err
	}

	file, _, err := r.FormFile(fileKey)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	csvLines, readErr := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if readErr != nil {
		return nil, readErr
	}

	req := convertCSVReq{
		key:      apiutil.ExtractThingKey(r),
		csvLines: csvLines,
		to:       r.URL.Query().Get("to"),
	}

	return req, nil
}

func decodeConvertJSONFile(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), multiPartContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		return nil, err
	}

	file, _, err := r.FormFile(fileKey)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	req := convertJSONReq{
		key:     apiutil.ExtractThingKey(r),
		records: records,
		to:      r.URL.Query().Get("to"),
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ any) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, messaging.ErrMalformedSubtopic):
		w.WriteHeader(http.StatusBadRequest)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
