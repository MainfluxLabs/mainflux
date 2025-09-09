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
		decodeListJSONMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/senml", kithttp.NewServer(
		listSenMLMessagesEndpoint(svc),
		decodeListSenMLMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/json", kithttp.NewServer(
		deleteJSONMessagesEndpoint(svc),
		decodeDeleteJSONMessages,
		encodeResponse,
		opts...,
	))

	mux.Delete("/senml", kithttp.NewServer(
		deleteSenMLMessagesEndpoint(svc),
		decodeDeleteSenMLMessages,
		encodeResponse,
		opts...,
	))

	mux.Get("/json/backup", kithttp.NewServer(
		backupJSONMessagesEndpoint(svc),
		decodeBackupJSONMessages,
		encodeBackupFileResponse,
		opts...,
	))

	mux.Get("/senml/backup", kithttp.NewServer(
		backupSenMLMessagesEndpoint(svc),
		decodeBackupSenMLMessages,
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

func decodeListJSONMessages(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
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

	return listJSONMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		pageMeta: readers.JSONMetadata{
			Offset:      offset,
			Limit:       limit,
			Subtopic:    subtopic,
			Protocol:    protocol,
			From:        from,
			To:          to,
			AggInterval: ai,
			AggType:     at,
			AggField:    af,
		},
	}, nil
}

func decodeListSenMLMessages(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadUintQuery(r, apiutil.OffsetKey, apiutil.DefOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadLimitQuery(r, apiutil.LimitKey, apiutil.DefLimit)
	if err != nil {
		return nil, err
	}

	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return nil, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, err
	}

	v, err := apiutil.ReadFloatQuery(r, valueKey, 0)
	if err != nil {
		return nil, err
	}

	comparator, err := apiutil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return nil, err
	}

	vs, err := apiutil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return nil, err
	}

	vd, err := apiutil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return nil, err
	}

	from, err := apiutil.ReadIntQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, toKey, 0)
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

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return nil, err
	}

	return listSenMLMessagesReq{
		token: apiutil.ExtractBearerToken(r),
		key:   apiutil.ExtractThingKey(r),
		pageMeta: readers.SenMLMetadata{
			Offset:      offset,
			Limit:       limit,
			Subtopic:    subtopic,
			Protocol:    protocol,
			Value:       v,
			Name:        name,
			StringValue: vs,
			DataValue:   vd,
			BoolValue:   vb,
			Comparator:  comparator,
			From:        from,
			To:          to,
			AggInterval: ai,
			AggType:     at,
			AggField:    af,
		},
	}, nil
}

func decodeDeleteJSONMessages(_ context.Context, r *http.Request) (interface{}, error) {
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
		key:   apiutil.ExtractThingKey(r),
		pageMeta: readers.JSONMetadata{
			From: from,
			To:   to,
		},
	}

	return req, nil
}

func decodeDeleteSenMLMessages(_ context.Context, r *http.Request) (interface{}, error) {
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
		key:   apiutil.ExtractThingKey(r),
		pageMeta: readers.SenMLMetadata{
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

func decodeBackupJSONMessages(_ context.Context, r *http.Request) (interface{}, error) {
	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
	if err != nil {
		return nil, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, apiutil.SubtopicKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := apiutil.ReadStringQuery(r, apiutil.ProtocolKey, "")
	if err != nil {
		return nil, err
	}

	from, err := apiutil.ReadIntQuery(r, apiutil.FromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, apiutil.ToKey, 0)
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

	return backupJSONMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		convertFormat: convertFormat,
		pageMeta: readers.JSONMetadata{
			Subtopic:    subtopic,
			Protocol:    protocol,
			From:        from,
			To:          to,
			AggInterval: ai,
			AggType:     at,
			AggField:    af,
		},
	}, nil
}

func decodeBackupSenMLMessages(_ context.Context, r *http.Request) (interface{}, error) {
	convertFormat, err := apiutil.ReadStringQuery(r, convertKey, jsonFormat)
	if err != nil {
		return nil, err
	}

	name, err := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	if err != nil {
		return nil, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, apiutil.SubtopicKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := apiutil.ReadStringQuery(r, apiutil.ProtocolKey, "")
	if err != nil {
		return nil, err
	}

	v, err := apiutil.ReadFloatQuery(r, apiutil.ValueKey, 0)
	if err != nil {
		return nil, err
	}

	comparator, err := apiutil.ReadStringQuery(r, apiutil.ComparatorKey, "")
	if err != nil {
		return nil, err
	}

	vs, err := apiutil.ReadStringQuery(r, apiutil.StringValueKey, "")
	if err != nil {
		return nil, err
	}

	vd, err := apiutil.ReadStringQuery(r, apiutil.DataValueKey, "")
	if err != nil {
		return nil, err
	}

	vb, err := apiutil.ReadBoolQuery(r, apiutil.BoolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return nil, err
	}

	from, err := apiutil.ReadIntQuery(r, apiutil.FromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadIntQuery(r, apiutil.ToKey, 0)
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

	return backupSenMLMessagesReq{
		token:         apiutil.ExtractBearerToken(r),
		convertFormat: convertFormat,
		pageMeta: readers.SenMLMetadata{
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
			AggType:     at,
			AggField:    af,
		},
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
