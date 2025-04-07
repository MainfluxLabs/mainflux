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
	"github.com/gofrs/uuid"
)

const (
	OffsetKey   = "offset"
	LimitKey    = "limit"
	NameKey     = "name"
	OrderKey    = "order"
	DirKey      = "dir"
	MetadataKey = "metadata"
	IDKey       = "id"

	NameOrder       = "name"
	IDOrder         = "id"
	AscDir          = "asc"
	DescDir         = "desc"
	ContentTypeJSON = "application/json"

	DefOffset = 0
	DefLimit  = 10
)

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64                 `json:"offset,omitempty"`
	Limit    uint64                 `json:"limit,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Order    string                 `json:"order,omitempty"`
	Dir      string                 `json:"dir,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger logger.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		switch {
		case errors.Contains(err, ErrBearerToken),
			errors.Contains(err, ErrMissingThingID),
			errors.Contains(err, ErrMissingProfileID),
			errors.Contains(err, ErrMissingGroupID),
			errors.Contains(err, ErrMissingMemberID),
			errors.Contains(err, ErrMissingOrgID),
			errors.Contains(err, ErrMissingWebhookID),
			errors.Contains(err, ErrMissingNotifierID),
			errors.Contains(err, ErrMissingAlarmID),
			errors.Contains(err, ErrMissingUserID),
			errors.Contains(err, ErrMissingRole),
			errors.Contains(err, ErrInvalidSubject),
			errors.Contains(err, ErrMissingObject),
			errors.Contains(err, ErrMissingKeyID),
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
			errors.Contains(err, ErrMissingCertID),
			errors.Contains(err, ErrMissingCertData),
			errors.Contains(err, ErrInvalidTopic),
			errors.Contains(err, ErrInvalidContact),
			errors.Contains(err, ErrMissingEmail),
			errors.Contains(err, ErrMissingHost),
			errors.Contains(err, ErrMissingPass),
			errors.Contains(err, ErrMissingConfPass),
			errors.Contains(err, ErrInvalidResetPass),
			errors.Contains(err, ErrInvalidComparator),
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

func EncodeError(err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, errors.ErrAuthentication):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrCreateEntity),
		errors.Contains(err, errors.ErrUpdateEntity),
		errors.Contains(err, errors.ErrRetrieveEntity),
		errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func WriteErrorResponse(err error, w http.ResponseWriter) {
	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", ContentTypeJSON)
		if err := json.NewEncoder(w).Encode(ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
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

func BuildPageMetadata(r *http.Request) (PageMetadata, error) {
	o, err := ReadUintQuery(r, OffsetKey, DefOffset)
	if err != nil {
		return PageMetadata{}, err
	}

	l, err := ReadLimitQuery(r, LimitKey, DefLimit)
	if err != nil {
		return PageMetadata{}, err
	}

	n, err := ReadStringQuery(r, NameKey, "")
	if err != nil {
		return PageMetadata{}, err
	}

	or, err := ReadStringQuery(r, OrderKey, IDOrder)
	if err != nil {
		return PageMetadata{}, err
	}

	d, err := ReadStringQuery(r, DirKey, DescDir)
	if err != nil {
		return PageMetadata{}, err
	}

	m, err := ReadMetadataQuery(r, MetadataKey, nil)
	if err != nil {
		return PageMetadata{}, err
	}

	return PageMetadata{
		Offset:   o,
		Limit:    l,
		Name:     n,
		Order:    or,
		Dir:      d,
		Metadata: m,
	}, nil
}

func ValidatePageMetadata(pm PageMetadata, maxLimitSize, maxNameSize int) error {
	if pm.Limit > uint64(maxLimitSize) {
		return ErrLimitSize
	}

	if len(pm.Name) > maxNameSize {
		return ErrNameSize
	}

	if pm.Order != "" &&
		pm.Order != NameOrder && pm.Order != IDOrder {
		return ErrInvalidOrder
	}

	if pm.Dir != "" &&
		pm.Dir != AscDir && pm.Dir != DescDir {
		return ErrInvalidDirection
	}

	return nil
}

func ValidateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return ErrInvalidIDFormat
	}

	return nil
}
