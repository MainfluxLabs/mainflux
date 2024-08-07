// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
)

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger logger.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		switch {
		case errors.Contains(err, ErrBearerToken),
			errors.Contains(err, ErrMissingID),
			errors.Contains(err, ErrMissingRole),
			errors.Contains(err, ErrInvalidSubject),
			errors.Contains(err, ErrMissingObject),
			errors.Contains(err, ErrInvalidAction),
			errors.Contains(err, ErrBearerKey),
			errors.Contains(err, ErrInvalidAuthKey),
			errors.Contains(err, ErrInvalidIDFormat),
			errors.Contains(err, ErrNameSize),
			errors.Contains(err, ErrLimitSize),
			errors.Contains(err, ErrOffsetSize),
			errors.Contains(err, ErrInvalidOrder),
			errors.Contains(err, ErrInvalidDirection),
			errors.Contains(err, ErrEmptyList),
			errors.Contains(err, ErrMissingCertData),
			errors.Contains(err, ErrInvalidTopic),
			errors.Contains(err, ErrInvalidContact),
			errors.Contains(err, ErrMissingEmail),
			errors.Contains(err, ErrMissingHost),
			errors.Contains(err, ErrMissingPass),
			errors.Contains(err, ErrMissingConfPass),
			errors.Contains(err, ErrInvalidResetPass),
			errors.Contains(err, ErrInvalidComparator),
			errors.Contains(err, ErrMissingMemberType),
			errors.Contains(err, ErrInvalidAPIKey),
			errors.Contains(err, ErrMaxLevelExceeded),
			errors.Contains(err, ErrUnsupportedContentType),
			errors.Contains(err, ErrMalformedEntity),
			errors.Contains(err, ErrInvalidRole),
			errors.Contains(err, ErrInvalidQueryParams):
			logger.Error(err.Error())
		}

		enc(ctx, err, w)
	}
}

// ReadUintQuery reads the value of uint64 http query parameters for a given key
func ReadUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, ErrInvalidQueryParams
	}

	return val, nil
}

// ReadIntQuery reads the value of int64 http query parameters for a given key
func ReadIntQuery(r *http.Request, key string, def int64) (int64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseInt(strval, 10, 64)
	if err != nil {
		return 0, ErrInvalidQueryParams
	}

	return val, nil
}

// ReadLimitQuery reads the value of limit http query parameters
func ReadLimitQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseInt(strval, 10, 64)
	if err != nil {
		return 0, ErrInvalidQueryParams
	}

	if val < -1 || val == 0 {
		return 0, ErrInvalidQueryParams
	}

	if val == -1 {
		val = 0
	}

	return uint64(val), nil
}

// ReadStringQuery reads the value of string http query parameters for a given key
func ReadStringQuery(r *http.Request, key string, def string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	return vals[0], nil
}

// ReadMetadataQuery reads the value of json http query parameters for a given key
func ReadMetadataQuery(r *http.Request, key string, def map[string]interface{}) (map[string]interface{}, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidQueryParams, err)
	}

	return m, nil
}

// ReadBoolQuery reads boolean query parameters in a given http request
func ReadBoolQuery(r *http.Request, key string, def bool) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return false, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, ErrInvalidQueryParams
	}

	return b, nil
}

// ReadFloatQuery reads the value of float64 http query parameters for a given key
func ReadFloatQuery(r *http.Request, key string, def float64) (float64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	fval := vals[0]
	val, err := strconv.ParseFloat(fval, 64)
	if err != nil {
		return 0, ErrInvalidQueryParams
	}

	return val, nil
}
