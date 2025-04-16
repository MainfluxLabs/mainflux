package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for Rule API endpoints.
func MakeHandler(tracer opentracing.Tracer, svc rules.Service, logger log.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/profiles/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_rules")(createRulesEndpoint(svc)),
		decodeCreateRules,
		encodeResponse,
		opts...,
	))
	r.Get("/rules/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_rule")(viewRuleEndpoint(svc)),
		decodeViewRule,
		encodeResponse,
		opts...,
	))
	r.Get("/profiles/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_rules_by_profile")(listRulesByProfileEndpoint(svc)),
		decodeListRulesByProfile,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_rules_by_group")(listRulesByGroupEndpoint(svc)),
		decodeListRulesByGroup,
		encodeResponse,
		opts...,
	))
	r.Put("/rules/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_rule")(updateRuleEndpoint(svc)),
		decodeUpdateRule,
		encodeResponse,
		opts...,
	))
	r.Patch("/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_rules")(removeRulesEndpoint(svc)),
		decodeRemoveRules,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("rules"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateRules(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createRulesReq{
		token:     apiutil.ExtractBearerToken(r),
		profileID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.Rules); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListRulesByProfile(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listRulesByProfileReq{
		token:        apiutil.ExtractBearerToken(r),
		profileID:    bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}
	return req, nil
}

func decodeListRulesByGroup(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listRulesByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeViewRule(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewRuleReq{token: apiutil.ExtractBearerToken(r), id: bone.GetValue(r, apiutil.IDKey)}

	return req, nil
}

func decodeUpdateRule(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateRuleReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req.ruleReq); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveRules(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeRulesReq{token: apiutil.ExtractBearerToken(r)}
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
		err == apiutil.ErrNameSize,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrMissingRuleID,
		err == apiutil.ErrMissingProfileID,
		err == apiutil.ErrMissingGroupID,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrMissingConditionField,
		err == apiutil.ErrMissingConditionOperator,
		err == apiutil.ErrMissingConditionThreshold,
		err == apiutil.ErrMissingActionID,
		err == apiutil.ErrInvalidActionType:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
