package orgs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/opentracing/opentracing-go"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer, logger logger.Logger) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}
	mux.Post("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_orgs")(createOrgsEndpoint(svc)),
		decodeCreateOrgs,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_org")(viewOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_org")(updateOrgEndpoint(svc)),
		decodeUpdateOrg,
		encodeResponse,
		opts...,
	))

	mux.Delete("/orgs/:id", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_org")(deleteOrgEndpoint(svc)),
		decodeOrgRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/orgs", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_orgs")(listOrgsEndpoint(svc)),
		decodeListOrgs,
		encodeResponse,
		opts...,
	))

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

func decodeListOrgs(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return nil, err
	}

	req := listOrgsReq{
		token:        apiutil.ExtractBearerToken(r),
		pageMetadata: pm,
	}

	return req, nil
}

func decodeCreateOrgs(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := createOrgsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUpdateOrg(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), apiutil.ContentTypeJSON) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateOrgReq{
		id:    bone.GetValue(r, apiutil.IDKey),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeOrgRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := orgReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, apiutil.IDKey),
	}

	return req, nil
}

func decodeBackup(_ context.Context, r *http.Request) (interface{}, error) {
	req := backupReq{
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func decodeRestore(_ context.Context, r *http.Request) (interface{}, error) {
	req := restoreReq{
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
	case errors.Contains(err, apiutil.ErrMalformedEntity),
		err == apiutil.ErrMissingOrgID,
		err == apiutil.ErrEmptyList,
		err == apiutil.ErrNameSize,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidOrder,
		err == apiutil.ErrInvalidDirection,
		err == apiutil.ErrInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, auth.ErrOrgNotEmpty):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, uuid.ErrGeneratingID):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		apiutil.EncodeError(err, w)
	}

	apiutil.WriteErrorResponse(err, w)
}
