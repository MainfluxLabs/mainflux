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
		listJSONMessagesEndpoint(svc),
		decodeListAllMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/senml", kithttp.NewServer(
		listSenMLMessagesEndpoint(svc),
		decodeListAllMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/json", kithttp.NewServer(
		deleteJSONMessagesEndpoint(svc),
		decodeDeleteMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/senml", kithttp.NewServer(
		deleteSenMLMessagesEndpoint(svc),
		decodeDeleteMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/json/backup", kithttp.NewServer(
		backupJSONMessagesEndpoint(svc),
		decodeBackupMessages,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Get("/senml/backup", kithttp.NewServer(
		backupSenMLMessagesEndpoint(svc),
		decodeBackupMessages,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Post("/json/restore", kithttp.NewServer(
		restoreJSONMessagesEndpoint(svc),
		decodeRestoreMessages,
		encodeResponse,
		opts...,
	))

	mux.Post("/senml/restore", kithttp.NewServer(
		restoreSenMLMessagesEndpoint(svc),
		decodeRestoreMessages,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListAllMessages(_ context.Context, r *http.Request) (interface{}, error) {
	pageMeta, err := BuildMessagePageMetadata(r)
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

	return listMessagesReq{
		token:    apiutil.ExtractBearerToken(r),
		key:      apiutil.ExtractThingKey(r),
		pageMeta: pageMeta,
	}, nil
}

func decodeDeleteMessages(_ context.Context, r *http.Request) (interface{}, error) {
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
			From: from,
			To:   to,
		},
	}

	return req, nil
}

func decodeRestoreMessages(_ context.Context, r *http.Request) (interface{}, error) {
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
		token:    apiutil.ExtractBearerToken(r),
		fileType: fileType,
		Messages: data,
	}, nil
}

func decodeBackupMessages(_ context.Context, r *http.Request) (interface{}, error) {
	pageMeta, err := BuildMessagePageMetadata(r)
	if err != nil {
		return nil, err
	}

	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
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

	return backupMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
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
	case errors.Contains(err, dbutil.ErrScanMetadata):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case errors.Contains(err, readers.ErrReadMessages):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
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

func BuildMessagePageMetadata(r *http.Request) (readers.PageMetadata, error) {
	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return readers.PageMetadata{}, err
	}

	limit, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return readers.PageMetadata{}, err
	}

	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	v, err := apiutil.ReadFloatQuery(r, valueKey, 0)
	if err != nil {
		return readers.PageMetadata{}, err
	}

	comparator, err := apiutil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	vs, err := apiutil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	vd, err := apiutil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return readers.PageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return readers.PageMetadata{}, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return readers.PageMetadata{}, err
	}

	pageMeta := readers.PageMetadata{
		Offset:      offset,
		Limit:       limit,
		Name:        name,
		Subtopic:    subtopic,
		Protocol:    protocol,
		Value:       v,
		Comparator:  comparator,
		StringValue: vs,
		DataValue:   vd,
		From:        from,
		To:          to,
	}

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return readers.PageMetadata{}, err
	}
	if err == nil {
		pageMeta.BoolValue = vb
	}

	return pageMeta, nil
}
