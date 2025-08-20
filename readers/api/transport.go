// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	auth "github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/readers"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	octetStreamContentType = "application/octet-stream"
	subtopicKey            = "subtopic"
	protocolKey            = "protocol"
	valueKey               = "v"
	stringValueKey         = "vs"
	dataValueKey           = "vd"
	boolValueKey           = "vb"
	comparatorKey          = "comparator"
	fromKey                = "from"
	toKey                  = "to"
	formatKey              = "format"
	convertKey             = "convert"
	aggIntervalKey         = "agg_interval"
	aggTypeKey             = "agg_type"
	aggFieldKey            = "agg_field"
	jsonFormat             = "json"
	senmlFormat            = "senml"
	csvFormat              = "csv"
	defFormat              = "messages"
)

var (
	thingc protomfx.ThingsServiceClient
	authc  protomfx.AuthServiceClient
)

func MakeHandler(svc readers.MessageRepository, tc protomfx.ThingsServiceClient, ac protomfx.AuthServiceClient, svcName string, logger logger.Logger) http.Handler {
	thingc = tc
	authc = ac

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()

	mux.Get("/json", kithttp.NewServer(
		listAllMessagesEndpoint(svc),
		decodeListAllMessagesJSON,
		encodeResponse,
		opts...,
	))

	mux.Get("/senml", kithttp.NewServer(
		listAllMessagesEndpoint(svc),
		decodeListAllMessagesSenML,
		encodeResponse,
		opts...,
	))

	mux.Delete("/json", kithttp.NewServer(
		deleteMessagesEndpoint(svc),
		decodeDeleteMessagesJSON,
		encodeResponse,
		opts...,
	))

	mux.Delete("/senml", kithttp.NewServer(
		deleteMessagesEndpoint(svc),
		decodeDeleteMessagesSenML,
		encodeResponse,
		opts...,
	))

	mux.Get("/json/backup", kithttp.NewServer(
		backupMessagesEndpoint(svc),
		decodeBackupMessagesJSON,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Get("/senml/backup", kithttp.NewServer(
		backupMessagesEndpoint(svc),
		decodeBackupMessagesSenML,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Post("/json/restore", kithttp.NewServer(
		restoreMessagesEndpoint(svc),
		decodeRestoreJSON,
		encodeResponse,
		opts...,
	))

	mux.Post("/senml/restore", kithttp.NewServer(
		restoreMessagesEndpoint(svc),
		decodeRestoreSenML,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListAllMessagesJSON(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeListAllMessagesWithFormat(ctx, r, jsonFormat)
}

func decodeListAllMessagesSenML(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeListAllMessagesWithFormat(ctx, r, senmlFormat)
}

func decodeListAllMessagesWithFormat(_ context.Context, r *http.Request, format string) (interface{}, error) {
	pageMeta, err := apiutil.BuildMessagePageMetadata(r)
	if err != nil {
		return nil, err
	}

	ai, err := apiutil.ReadStringQuery(r, aggIntervalKey, "")
	if err != nil {
		return nil, err
	}

	at, err := apiutil.ReadStringQuery(r, aggTypeKey, "")
	if err != nil {
		return nil, err
	}

	af, err := apiutil.ReadStringQuery(r, aggFieldKey, "")
	if err != nil {
		return nil, err
	}

	pageMeta.AggInterval = ai
	pageMeta.AggType = at
	pageMeta.AggField = af
	pageMeta.Format = dbutil.GetTableName(format)

	return listAllMessagesReq{
		token:    apiutil.ExtractBearerToken(r),
		key:      apiutil.ExtractThingKey(r),
		pageMeta: pageMeta,
	}, nil
}

func decodeDeleteMessagesJSON(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeDeleteMessagesWithFormat(ctx, r, jsonFormat)
}

func decodeDeleteMessagesSenML(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeDeleteMessagesWithFormat(ctx, r, senmlFormat)
}

func decodeDeleteMessagesWithFormat(_ context.Context, r *http.Request, format string) (interface{}, error) {
	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := deleteMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		key:   apiutil.ExtractThingKey(r),
		pageMeta: readers.PageMetadata{
			From:   from,
			To:     to,
			Format: dbutil.GetTableName(format),
		},
	}

	return req, nil
}

func decodeRestoreJSON(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeRestoreWithFormat(ctx, r, jsonFormat)
}

func decodeRestoreSenML(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeRestoreWithFormat(ctx, r, senmlFormat)
}

func decodeRestoreWithFormat(_ context.Context, r *http.Request, messageFormat string) (interface{}, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	var fileType string
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case apiutil.ContentTypeJSON:
		fileType = jsonFormat
	case apiutil.ContentTypeCSV:
		fileType = csvFormat
	default:
		return nil, errors.Wrap(apiutil.ErrUnsupportedContentType, err)
	}

	return restoreMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		fileType:      fileType,
		messageFormat: messageFormat,
		Messages:      data,
	}, nil
}

func decodeBackupMessagesJSON(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeBackupMessagesWithFormat(ctx, r, jsonFormat)
}

func decodeBackupMessagesSenML(ctx context.Context, r *http.Request) (interface{}, error) {
	return decodeBackupMessagesWithFormat(ctx, r, senmlFormat)
}

func decodeBackupMessagesWithFormat(_ context.Context, r *http.Request, format string) (interface{}, error) {
	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
	if err != nil {
		return nil, err
	}

	pageMeta, err := apiutil.BuildMessagePageMetadata(r)
	if err != nil {
		return nil, err
	}

	return backupMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		messageFormat: format,
		convertFormat: convertFormat,
		pageMeta:      pageMeta,
	}, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
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

func encodeBackupFileResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", octetStreamContentType)

	if ar, ok := response.(backupFileRes); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}

		w.Write(ar.file)
	}

	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, nil):
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrInvalidComparator:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, readers.ErrReadMessages),
		errors.Contains(err, errors.ErrCreateEntity):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	apiutil.WriteErrorResponse(err, w)
}

func getPubConfByKey(ctx context.Context, key string) (*protomfx.PubConfByKeyRes, error) {
	pc, err := thingc.GetPubConfByKey(ctx, &protomfx.PubConfByKeyReq{Key: key})
	if err != nil {
		return nil, err
	}

	return pc, nil
}

func isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: auth.RootSub,
	}

	if _, err := authc.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}
