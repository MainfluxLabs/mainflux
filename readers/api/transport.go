// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/things"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
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
	convertKey             = "convert"
	filterKey              = "filter"
	aggIntervalKey         = "agg_interval"
	aggValueKey            = "agg_value"
	aggTypeKey             = "agg_type"
	aggFieldKey            = "agg_field"
	publisherKey           = "publisher"
	publisherIDKey         = "publisherID"
	timeFormatKey          = "time_format"
	jsonFormat             = "json"
	csvFormat              = "csv"
)

func MakeHandler(svc readers.Service, tracer opentracing.Tracer, svcName string, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	mux := bone.New()

	mux.Get("/json", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_json_messages")(listJSONMessagesEndpoint(svc)),
		decodeListJSONMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/senml", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_senml_messages")(listSenMLMessagesEndpoint(svc)),
		decodeListSenMLMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/json", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_all_json_messages")(deleteAllJSONMessagesEndpoint(svc)),
		decodeDeleteAllJSONMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/json/:publisherID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_json_messages")(deleteJSONMessagesEndpoint(svc)),
		decodeDeleteJSONMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/senml", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_all_senml_messages")(deleteAllSenMLMessagesEndpoint(svc)),
		decodeDeleteAllSenMLMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/senml/:publisherID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_senml_messages")(deleteSenMLMessagesEndpoint(svc)),
		decodeDeleteSenMLMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/json/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_json_messages")(backupJSONMessagesEndpoint(svc)),
		decodeBackupJSONMessages,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Get("/senml/backup", kithttp.NewServer(
		kitot.TraceServer(tracer, "backup_senml_messages")(backupSenMLMessagesEndpoint(svc)),
		decodeBackupSenMLMessages,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Post("/json/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_json_messages")(restoreJSONMessagesEndpoint(svc)),
		decodeRestoreMessages,
		encodeResponse,
		opts...,
	))

	mux.Post("/senml/restore", kithttp.NewServer(
		kitot.TraceServer(tracer, "restore_senml_messages")(restoreSenMLMessagesEndpoint(svc)),
		decodeRestoreMessages,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListJSONMessages(_ context.Context, r *http.Request) (any, error) {
	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	pageMeta, err := BuildJSONPageMetadata(r)
	if err != nil {
		return nil, err
	}

	filter, err := apiutil.ReadStringQuery(r, filterKey, "")
	if err != nil {
		return nil, err
	}

	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	pageMeta.Offset = offset
	pageMeta.Limit = limit
	pageMeta.Publisher = publisher
	pageMeta.Filter = filter

	return listJSONMessagesReq{
		token:    apiutil.ExtractBearerToken(r),
		thingKey: things.ExtractThingKey(r),
		pageMeta: pageMeta,
	}, nil
}

func decodeListSenMLMessages(_ context.Context, r *http.Request) (any, error) {
	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	pageMeta, err := BuildSenMLPageMetadata(r)
	if err != nil {
		return nil, err
	}

	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	pageMeta.Offset = offset
	pageMeta.Limit = limit
	pageMeta.Publisher = publisher

	return listSenMLMessagesReq{
		token:    apiutil.ExtractBearerToken(r),
		thingKey: things.ExtractThingKey(r),
		pageMeta: pageMeta,
	}, nil
}

func decodeDeleteAllJSONMessages(_ context.Context, r *http.Request) (any, error) {
	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := deleteAllJSONMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMeta: readers.JSONPageMetadata{
			Subtopic: subtopic,
			Protocol: protocol,
			From:     from,
			To:       to,
		},
	}

	return req, nil
}

func decodeDeleteJSONMessages(_ context.Context, r *http.Request) (any, error) {
	publisherID := bone.GetValue(r, publisherIDKey)

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := deleteJSONMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMeta: readers.JSONPageMetadata{
			Publisher: publisherID,
			Subtopic:  subtopic,
			Protocol:  protocol,
			From:      from,
			To:        to,
		},
	}

	return req, nil
}

func decodeDeleteAllSenMLMessages(_ context.Context, r *http.Request) (any, error) {
	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := deleteAllSenMLMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMeta: readers.SenMLPageMetadata{
			From: from,
			To:   to,
		},
	}

	return req, nil
}

func decodeDeleteSenMLMessages(_ context.Context, r *http.Request) (any, error) {
	publisher := bone.GetValue(r, publisherIDKey)

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := deleteSenMLMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMeta: readers.SenMLPageMetadata{
			Publisher: publisher,
			Subtopic:  subtopic,
			Protocol:  protocol,
			From:      from,
			To:        to,
		},
	}

	return req, nil
}

func decodeRestoreMessages(_ context.Context, r *http.Request) (any, error) {
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

func decodeBackupJSONMessages(_ context.Context, r *http.Request) (any, error) {
	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
	if err != nil {
		return nil, err
	}

	timeFormat, err := apiutil.ReadStringQuery(r, timeFormatKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	pageMeta, err := BuildJSONPageMetadata(r)
	if err != nil {
		return nil, err
	}

	pageMeta.Publisher = publisher

	return backupJSONMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		convertFormat: convertFormat,
		timeFormat:    timeFormat,
		pageMeta:      pageMeta,
	}, nil
}

func decodeBackupSenMLMessages(_ context.Context, r *http.Request) (any, error) {
	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
	if err != nil {
		return nil, err
	}

	timeFormat, err := apiutil.ReadStringQuery(r, timeFormatKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	pageMeta, err := BuildSenMLPageMetadata(r)
	if err != nil {
		return nil, err
	}

	pageMeta.Publisher = publisher

	return backupSenMLMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		convertFormat: convertFormat,
		timeFormat:    timeFormat,
		pageMeta:      pageMeta,
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

func encodeBackupFileResponse(_ context.Context, w http.ResponseWriter, response any) error {
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

func BuildJSONPageMetadata(r *http.Request) (readers.JSONPageMetadata, error) {
	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	ai, err := apiutil.ReadStringQuery(r, aggIntervalKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	av, err := apiutil.ReadUintQuery(r, aggValueKey, 1)
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	at, err := apiutil.ReadStringQuery(r, aggTypeKey, "")
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	af, err := apiutil.ReadStringArrayQuery(r, aggFieldKey)
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	d, err := apiutil.ReadStringQuery(r, apiutil.DirKey, apiutil.DescDir)
	if err != nil {
		return readers.JSONPageMetadata{}, err
	}

	pageMeta := readers.JSONPageMetadata{
		Subtopic:    subtopic,
		Protocol:    protocol,
		From:        from,
		To:          to,
		AggInterval: ai,
		AggValue:    av,
		AggType:     at,
		AggFields:   af,
		Dir:         d,
	}

	return pageMeta, nil
}

func BuildSenMLPageMetadata(r *http.Request) (readers.SenMLPageMetadata, error) {
	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	v, err := apiutil.ReadFloatQuery(r, valueKey, 0)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	comparator, err := apiutil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	vs, err := apiutil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	vd, err := apiutil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return readers.SenMLPageMetadata{}, err
	}

	ai, err := apiutil.ReadStringQuery(r, aggIntervalKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	av, err := apiutil.ReadUintQuery(r, aggValueKey, 1)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	at, err := apiutil.ReadStringQuery(r, aggTypeKey, "")
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	af, err := apiutil.ReadStringArrayQuery(r, aggFieldKey)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	d, err := apiutil.ReadStringQuery(r, apiutil.DirKey, apiutil.DescDir)
	if err != nil {
		return readers.SenMLPageMetadata{}, err
	}

	pageMeta := readers.SenMLPageMetadata{
		Name:        name,
		Subtopic:    subtopic,
		Protocol:    protocol,
		Value:       v,
		Comparator:  comparator,
		StringValue: vs,
		DataValue:   vd,
		BoolValue:   vb,
		From:        from,
		To:          to,
		AggInterval: ai,
		AggValue:    av,
		AggType:     at,
		AggFields:   af,
		Dir:         d,
	}

	return pageMeta, nil
}
