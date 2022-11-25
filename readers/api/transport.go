// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType    = "application/json"
	offsetKey      = "offset"
	limitKey       = "limit"
	formatKey      = "format"
	subtopicKey    = "subtopic"
	publisherKey   = "publisher"
	protocolKey    = "protocol"
	nameKey        = "name"
	valueKey       = "v"
	stringValueKey = "vs"
	dataValueKey   = "vd"
	boolValueKey   = "vb"
	comparatorKey  = "comparator"
	fromKey        = "from"
	toKey          = "to"
	defLimit       = 10
	defOffset      = 0
	defFormat      = "messages"
)

var (
	errThingAccess = errors.New("thing has no permission")
	errUserAccess  = errors.New("user has no permission")
	thingsAuth     mainflux.ThingsServiceClient
	usersAuth      mainflux.AuthServiceClient
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient, svcName string, logger logger.Logger) http.Handler {
	thingsAuth = tc
	usersAuth = ac

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()
	mux.Get("/channels/:chanID/messages", kithttp.NewServer(
		ListChannelMessagesEndpoint(svc),
		decodeListMessages,
		encodeResponse,
		opts...,
	))
	mux.Get("/messages", kithttp.NewServer(
		listAllMessagesEndpoint(svc),
		decodeListMessages,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeListMessages(ctx context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	limit, err := apiutil.ReadIntQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	format, err := apiutil.ReadStringQuery(r, formatKey, defFormat)
	if err != nil {
		return nil, err
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, err
	}

	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, err
	}

	name, err := apiutil.ReadStringQuery(r, nameKey, "")
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

	from, err := apiutil.ReadFloatQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := apiutil.ReadFloatQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := listMessagesReq{
		chanID: bone.GetValue(r, "chanID"),
		token:  apiutil.ExtractBearerToken(r),
		key:    apiutil.ExtractThingKey(r),
		pageMeta: readers.PageMetadata{
			Offset:      offset,
			Limit:       limit,
			Format:      format,
			Subtopic:    subtopic,
			Publisher:   publisher,
			Protocol:    protocol,
			Name:        name,
			Value:       v,
			Comparator:  comparator,
			StringValue: vs,
			DataValue:   vd,
			From:        from,
			To:          to,
		},
	}

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != errors.ErrNotFoundParam {
		return nil, err
	}
	if err == nil {
		req.pageMeta.BoolValue = vb
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
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
	case errors.Contains(err, nil):
	case errors.Contains(err, errors.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrInvalidComparator:
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, readers.ErrReadMessages):
		w.WriteHeader(http.StatusInternalServerError)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func authorize(ctx context.Context, token, key, chanID string) (err error) {
	switch {
	case token != "":
		user, err := usersAuth.Identify(ctx, &mainflux.Token{Value: token})
		if err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		if _, err = thingsAuth.IsChannelOwner(ctx, &mainflux.ChannelOwnerReq{Owner: user.Email, ChanID: chanID}); err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		return nil
	default:
		if _, err := thingsAuth.CanAccessByKey(ctx, &mainflux.AccessByKeyReq{Token: key, ChanID: chanID}); err != nil {
			return errors.Wrap(errThingAccess, err)
		}
		return nil
	}
}

func authorizeAdmin(ctx context.Context, object, relation, token string) error {
	identity, err := usersAuth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	req := &mainflux.AuthorizeReq{
		Sub: identity.Id,
		Obj: object,
		Act: relation,
	}

	res, err := usersAuth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}

	return nil
}
