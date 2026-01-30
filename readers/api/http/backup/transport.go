// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

const octetStreamContentType = "application/octet-stream"

func MakeHandler(svc readers.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

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

func decodeBackup(_ context.Context, r *http.Request) (any, error) {
	return backupReq{
		token: apiutil.ExtractBearerToken(r),
	}, nil
}

func decodeRestore(_ context.Context, r *http.Request) (any, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	return restoreReq{
		token:    apiutil.ExtractBearerToken(r),
		Messages: data,
	}, nil
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

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, dbutil.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, readers.ErrReadMessages):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
