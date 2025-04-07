package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Alarm API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc alarms.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Get("/things/:id/alarms", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_alarms_by_thing")(listAlarmsByThingEndpoint(svc)),
		decodeListAlarmsByThing,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/alarms", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_alarms_by_group")(listAlarmsByGroupEndpoint(svc)),
		decodeListGroupAlarms,
		encodeResponse,
		opts...,
	))

	r.Get("/alarms/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_alarm")(viewAlarmEndpoint(svc)),
		decodeViewAlarm,
		encodeResponse,
		opts...,
	))

	r.Patch("/alarms", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_alarms")(removeAlarmsEndpoint(svc)),
		decodeRemoveAlarms,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("alarms"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeListAlarmsByThing(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	return listAlarmsByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}, nil
}

func decodeListGroupAlarms(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	return listAlarmsByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}, nil
}

func decodeViewAlarm(_ context.Context, r *http.Request) (interface{}, error) {
	return alarmReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}, nil
}

func decodeRemoveAlarms(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeAlarmsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
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

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {
	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingAlarmID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrMissingThingID:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
