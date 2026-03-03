// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	adapter "github.com/MainfluxLabs/mainflux/converters"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol             = "http"
	ctJSON               = "application/json"
	maxMemory            = 32 << 20
	fileKey              = "file"
	multiPartContentType = "multipart/form-data"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, tracer opentracing.Tracer, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/csv/senml", kithttp.NewServer(
		kitot.TraceServer(tracer, "convert_csv_to_senml")(convertCSVToSenMLEndpoint(svc)),
		decodeConvertCSVFile,
		encodeResponse,
		opts...,
	))

	r.Post("/csv/json", kithttp.NewServer(
		kitot.TraceServer(tracer, "convert_csv_to_json")(convertCSVToJSONEndpoint(svc)),
		decodeConvertCSVFile,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("converters"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

type SenmlRec struct {
	Name  string `json:"n"`
	Time  string `json:"t"`
	Value string `json:"v"`
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

	csvLines, readErr := csv.NewReader(file).ReadAll()
	if readErr != nil {
		return nil, err
	}

	req := convertCSVReq{
		key:      things.ExtractThingKey(r),
		csvLines: csvLines,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ any) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, messaging.ErrMalformedSubtopic),
		errors.Contains(err, apiutil.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", ctJSON)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
