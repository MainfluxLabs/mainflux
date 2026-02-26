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

	r.Post("/groups/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_rules")(createRulesEndpoint(svc)),
		decodeCreateRules,
		encodeResponse,
		opts...,
	))
	r.Get("/rules/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_rule")(viewRuleEndpoint(svc)),
		decodeRuleReq,
		encodeResponse,
		opts...,
	))
	r.Get("/things/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_rules_by_thing")(listRulesByThingEndpoint(svc)),
		decodeListRulesByThing,
		encodeResponse,
		opts...,
	))
	r.Get("/groups/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_rules_by_group")(listRulesByGroupEndpoint(svc)),
		decodeListRulesByGroup,
		encodeResponse,
		opts...,
	))
	r.Get("/rules/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_thing_ids_by_rule")(listThingIDsByRuleEndpoint(svc)),
		decodeRuleReq,
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
	r.Post("/things/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_rules")(assignRulesEndpoint(svc)),
		decodeThingRules,
		encodeResponse,
		opts...,
	))
	r.Patch("/things/:id/rules", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_rules")(unassignRulesEndpoint(svc)),
		decodeThingRules,
		encodeResponse,
		opts...,
	))

	r.Post("/groups/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_scripts")(createScriptsEndpoint(svc)),
		decodeCreateScripts,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_scripts_by_thing")(listScriptsByThingEndpoint(svc)),
		decodeListScriptsByThing,
		encodeResponse,
		opts...,
	))

	r.Get("/groups/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_scripts_by_group")(listScriptsByGroupEndpoint(svc)),
		decodeListScriptsByGroup,
		encodeResponse,
		opts...,
	))

	r.Get("/scripts/:id/things", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_thing_ids_by_script")(listThingIDsByScriptEndpoint(svc)),
		decodeScriptReq,
		encodeResponse,
		opts...,
	))

	r.Get("/scripts/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_script")(viewScriptEndpoint(svc)),
		decodeScriptReq,
		encodeResponse,
		opts...,
	))

	r.Put("/scripts/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_script")(updateScriptEndpoint(svc)),
		decodeUpdateScript,
		encodeResponse,
		opts...,
	))

	r.Patch("/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_scripts")(removeScriptsEndpoint(svc)),
		decodeRemoveScripts,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_scripts")(assignScriptsEndpoint(svc)),
		decodeModifyThingScripts,
		encodeResponse,
		opts...,
	))

	r.Patch("/things/:id/scripts", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign_scripts")(unassignScriptsEndpoint(svc)),
		decodeModifyThingScripts,
		encodeResponse,
		opts...,
	))

	r.Get("/things/:id/runs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_script_runs_by_thing")(listScriptRunsByThingEndpoint(svc)),
		decodeListScriptRunsByThing,
		encodeResponse,
		opts...,
	))

	r.Patch("/runs", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_script_runs")(removeScriptRunsEndpoint(svc)),
		decodeRemoveScriptRuns,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("rules"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeCreateRules(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createRulesReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListRulesByThing(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listRulesByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}
	return req, nil
}

func decodeListRulesByGroup(_ context.Context, r *http.Request) (any, error) {
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

func decodeRuleReq(_ context.Context, r *http.Request) (any, error) {
	req := ruleReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeUpdateRule(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateRuleReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveRules(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeRulesReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}
func decodeThingRules(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := thingRulesReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeCreateScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createScriptsReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScriptsByThing(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listScriptsByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}
	return req, nil
}

func decodeListScriptsByGroup(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listScriptsByGroupReq{
		token:        apiutil.ExtractBearerToken(r),
		groupID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}
	return req, nil
}

func decodeScriptReq(_ context.Context, r *http.Request) (any, error) {
	req := scriptReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeUpdateScript(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateScriptReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeScriptsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeModifyThingScripts(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := thingScriptsReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListScriptRunsByThing(_ context.Context, r *http.Request) (any, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listScriptRunsByThingReq{
		token:        apiutil.ExtractBearerToken(r),
		thingID:      bone.GetValue(r, apiutil.IDKey),
		pageMetadata: pm,
	}
	return req, nil
}

func decodeRemoveScriptRuns(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeScriptRunsReq{
		token: apiutil.ExtractBearerToken(r),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
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
	case errors.Contains(err, rules.ErrScriptSize):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
