// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for rule API endpoints.
func MakeHandler(svc rules.Service, ac domain.AuthClient, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
		kithttp.ServerBefore(authn.HTTPTokenToContext),
	}

	withIdentity := authn.IdentityMiddleware(ac, logger)

	mux.Post("/groups/:id/rules", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "create_rules"),
			withIdentity,
		)(createRulesEndpoint(svc)),
		decodeCreateRules,
		encodeResponse,
		opts...,
	))
	mux.Get("/rules/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "view_rule"),
			withIdentity,
		)(viewRuleEndpoint(svc)),
		decodeRuleReq,
		encodeResponse,
		opts...,
	))
	mux.Get("/things/:id/rules", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_rules_by_thing"),
			withIdentity,
		)(listRulesByThingEndpoint(svc)),
		decodeListRulesByThing,
		encodeResponse,
		opts...,
	))
	mux.Get("/groups/:id/rules", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_rules_by_group"),
			withIdentity,
		)(listRulesByGroupEndpoint(svc)),
		decodeListRulesByGroup,
		encodeResponse,
		opts...,
	))
	mux.Get("/rules/:id/things", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "list_thing_ids_by_rule"),
			withIdentity,
		)(listThingIDsByRuleEndpoint(svc)),
		decodeRuleReq,
		encodeResponse,
		opts...,
	))
	mux.Put("/rules/:id", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "update_rule"),
			withIdentity,
		)(updateRuleEndpoint(svc)),
		decodeUpdateRule,
		encodeResponse,
		opts...,
	))
	mux.Post("/rules/:id/things", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "assign_things"),
			withIdentity,
		)(assignThingsEndpoint(svc)),
		decodeRuleThings,
		encodeResponse,
		opts...,
	))
	mux.Patch("/rules/:id/things", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "unassign_things"),
			withIdentity,
		)(unassignThingsEndpoint(svc)),
		decodeRuleThings,
		encodeResponse,
		opts...,
	))
	mux.Patch("/rules", kithttp.NewServer(
		endpoint.Chain(
			kitot.TraceServer(tracer, "remove_rules"),
			withIdentity,
		)(removeRulesEndpoint(svc)),
		decodeRemoveRules,
		encodeResponse,
		opts...,
	))
	return mux
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeListRulesByThing(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	name, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	inputType, _ := apiutil.ReadStringQuery(r, apiutil.InputTypeKey, "")

	req := listRulesByThingReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: rules.PageMetadata{
			Offset:    base.Offset,
			Limit:     base.Limit,
			Order:     base.Order,
			Dir:       base.Dir,
			Name:      name,
			InputType: inputType,
		},
	}

	return req, nil
}

func decodeListRulesByGroup(_ context.Context, r *http.Request) (any, error) {
	base, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}
	name, _ := apiutil.ReadStringQuery(r, apiutil.NameKey, "")
	inputType, _ := apiutil.ReadStringQuery(r, apiutil.InputTypeKey, "")

	req := listRulesByGroupReq{
		token:   apiutil.ExtractBearerToken(r),
		groupID: bone.GetValue(r, apiutil.IDKey),
		pageMetadata: rules.PageMetadata{
			Offset:    base.Offset,
			Limit:     base.Limit,
			Order:     base.Order,
			Dir:       base.Dir,
			Name:      name,
			InputType: inputType,
		},
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
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRuleThings(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := ruleThingsReq{
		token:  apiutil.ExtractBearerToken(r),
		ruleID: bone.GetValue(r, apiutil.IDKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeRemoveRules(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := removeRulesReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
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
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
