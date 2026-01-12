// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	OffsetKey              = "offset"
	LimitKey               = "limit"
	NameKey                = "name"
	OrderKey               = "order"
	DirKey                 = "dir"
	MetadataKey            = "metadata"
	IDKey                  = "id"
	SerialKey              = "serial"
	EmailKey               = "email"
	PayloadKey             = "payload"
	SubtopicKey            = "subtopic"
	ProtocolKey            = "protocol"
	ValueKey               = "v"
	StringValueKey         = "vs"
	DataValueKey           = "vd"
	BoolValueKey           = "vb"
	ComparatorKey          = "comparator"
	FromKey                = "from"
	ToKey                  = "to"
	NameOrder              = "name"
	IDOrder                = "id"
	AscDir                 = "asc"
	DescDir                = "desc"
	ContentTypeJSON        = "application/json"
	ContentTypeCSV         = "text/csv"
	ContentTypeOctetStream = "application/octet-stream"
	DefOffset              = 0
	DefLimit               = 10
)

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64         `json:"offset,omitempty"`
	Limit    uint64         `json:"limit,omitempty"`
	Name     string         `json:"name,omitempty"`
	Order    string         `json:"order,omitempty"`
	Dir      string         `json:"dir,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Email    string         `json:"email,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
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
			errors.Contains(err, ErrMissingNotifierID),
			errors.Contains(err, ErrMissingAlarmID),
			errors.Contains(err, ErrMissingUserID),
			errors.Contains(err, ErrMissingRole),
			errors.Contains(err, ErrInvalidSubject),
			errors.Contains(err, ErrMissingObject),
			errors.Contains(err, ErrMissingKeyID),
			errors.Contains(err, ErrMissingInviteID),
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
			errors.Contains(err, ErrMissingSerial),
			errors.Contains(err, ErrMissingCertData),
			errors.Contains(err, ErrInvalidContact),
			errors.Contains(err, ErrMissingEmail),
			errors.Contains(err, ErrMissingPass),
			errors.Contains(err, ErrMissingRedirectPath),
			errors.Contains(err, ErrMissingConfPass),
			errors.Contains(err, ErrInvalidResetPass),
			errors.Contains(err, ErrInvalidComparator),
			errors.Contains(err, ErrInvalidAPIKey),
			errors.Contains(err, ErrUnsupportedContentType),
			errors.Contains(err, ErrMalformedEntity),
			errors.Contains(err, ErrInvalidRole),
			errors.Contains(err, ErrInvalidQueryParams),
			errors.Contains(err, ErrMissingConditionField),
			errors.Contains(err, ErrMissingConditionComparator),
			errors.Contains(err, ErrMissingConditionThreshold),
			errors.Contains(err, ErrInvalidActionType),
			errors.Contains(err, ErrMissingActionID),
			errors.Contains(err, ErrInvalidOperator):
			logger.Error(err.Error())
		}

		enc(ctx, err, w)
	}
}

// Map a gRPC-error Status code to an HTTP status code and write it to w
func EncodeGRPCError(st *status.Status, w http.ResponseWriter) {
	switch st.Code() {
	case codes.OK:
		w.WriteHeader(http.StatusOK)
	case codes.Internal:
		w.WriteHeader(http.StatusInternalServerError)
	case codes.InvalidArgument:
		w.WriteHeader(http.StatusBadRequest)
	case codes.Unauthenticated:
		w.WriteHeader(http.StatusUnauthorized)
	case codes.NotFound:
		w.WriteHeader(http.StatusNotFound)
	case codes.AlreadyExists:
		w.WriteHeader(http.StatusConflict)
	case codes.PermissionDenied:
		w.WriteHeader(http.StatusForbidden)
	case codes.Canceled:
		w.WriteHeader(http.StatusRequestTimeout)
	case codes.DeadlineExceeded:
		w.WriteHeader(http.StatusGatewayTimeout)
	case codes.ResourceExhausted:
		w.WriteHeader(http.StatusTooManyRequests)
	case codes.FailedPrecondition:
		w.WriteHeader(http.StatusPreconditionFailed)
	case codes.Aborted:
		w.WriteHeader(http.StatusConflict)
	case codes.OutOfRange:
		w.WriteHeader(http.StatusBadRequest)
	case codes.Unimplemented:
		w.WriteHeader(http.StatusNotImplemented)
	case codes.Unavailable:
		w.WriteHeader(http.StatusServiceUnavailable)
	case codes.DataLoss:
		w.WriteHeader(http.StatusInternalServerError)
	case codes.Unknown:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func EncodeError(err error, w http.ResponseWriter) {
	switch {
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, ErrBearerToken),
		errors.Contains(err, ErrBearerKey),
		errors.Contains(err, ErrInvalidThingKeyType),
		errors.Contains(err, ErrMissingExternalThingKey):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, ErrMissingGroupID),
		errors.Contains(err, ErrMissingOrgID),
		errors.Contains(err, ErrMissingThingID),
		errors.Contains(err, ErrMissingProfileID),
		errors.Contains(err, ErrMissingMemberID),
		errors.Contains(err, ErrMissingNotifierID),
		errors.Contains(err, ErrMissingAlarmID),
		errors.Contains(err, ErrMissingRuleID),
		errors.Contains(err, ErrMissingUserID),
		errors.Contains(err, ErrMissingRole),
		errors.Contains(err, ErrMissingObject),
		errors.Contains(err, ErrMissingKeyID),
		errors.Contains(err, ErrMissingInviteID),
		errors.Contains(err, ErrInvalidIDFormat),
		errors.Contains(err, ErrNameSize),
		errors.Contains(err, ErrEmailSize),
		errors.Contains(err, ErrInvalidStatus),
		errors.Contains(err, ErrLimitSize),
		errors.Contains(err, ErrOffsetSize),
		errors.Contains(err, ErrInvalidOrder),
		errors.Contains(err, ErrInvalidDirection),
		errors.Contains(err, ErrEmptyList),
		errors.Contains(err, ErrMissingSerial),
		errors.Contains(err, ErrMissingCertData),
		errors.Contains(err, ErrInvalidContact),
		errors.Contains(err, ErrMissingEmail),
		errors.Contains(err, ErrMissingEmailToken),
		errors.Contains(err, ErrMissingRedirectPath),
		errors.Contains(err, ErrMissingPass),
		errors.Contains(err, ErrMissingConfPass),
		errors.Contains(err, ErrInvalidResetPass),
		errors.Contains(err, ErrInvalidComparator),
		errors.Contains(err, ErrInvalidAPIKey),
		errors.Contains(err, ErrInvalidQueryParams),
		errors.Contains(err, ErrInvalidAggType),
		errors.Contains(err, ErrNotFoundParam),
		errors.Contains(err, ErrMalformedEntity),
		errors.Contains(err, ErrInvalidRole),
		errors.Contains(err, ErrMissingConditionField),
		errors.Contains(err, ErrMissingConditionThreshold),
		errors.Contains(err, ErrInvalidActionType),
		errors.Contains(err, ErrMissingActionID),
		errors.Contains(err, ErrInvalidOperator):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthorization),
		errors.Contains(err, ErrInviteExpired),
		errors.Contains(err, ErrInvalidInviteState):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, dbutil.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, dbutil.ErrConflict),
		errors.Contains(err, ErrUserAlreadyInvited):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, dbutil.ErrCreateEntity),
		errors.Contains(err, dbutil.ErrUpdateEntity),
		errors.Contains(err, dbutil.ErrRetrieveEntity),
		errors.Contains(err, dbutil.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		if st, ok := status.FromError(err); ok {
			EncodeGRPCError(st, w)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}
}

func WriteErrorResponse(err error, w http.ResponseWriter) {
	var errorMessage string

	switch e := err.(type) {
	case errors.Error:
		errorMessage = e.Msg()
	default:
		if st, ok := status.FromError(err); ok {
			// Cut the error message short to avoid exposing details of wrapped errors
			errorMessage, _, _ = strings.Cut(st.Message(), " :")
		}
	}

	if errorMessage != "" {
		w.Header().Set("Content-Type", ContentTypeJSON)
		if err := json.NewEncoder(w).Encode(ErrorRes{Err: errorMessage}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func EncodeFileResponse(_ context.Context, w http.ResponseWriter, response any) (err error) {
	w.Header().Set("Content-Type", ContentTypeOctetStream)

	if fr, ok := response.(ViewFileRes); ok {
		for k, v := range fr.Headers() {
			w.Header().Set(k, v)
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fr.FileName))
		w.WriteHeader(fr.Code())

		if fr.Empty() {
			return nil
		}

		_, err := w.Write(fr.File)
		return err
	}

	return nil
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

	if val <= 0 {
		return 0, ErrInvalidQueryParams
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
func ReadMetadataQuery(r *http.Request, key string, def map[string]any) (map[string]any, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	m := make(map[string]any)
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

func ReadStringArrayQuery(r *http.Request, key string) ([]string, error) {
	vals := bone.GetQuery(r, key)

	if len(vals) > 10 {
		return nil, ErrInvalidQueryParams
	}

	return vals, nil
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

	e, err := ReadStringQuery(r, EmailKey, "")
	if err != nil {
		return PageMetadata{}, err
	}

	p, err := ReadMetadataQuery(r, PayloadKey, nil)
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
		Email:    e,
		Payload:  p,
	}, nil
}

func BuildPageMetadataFromBody(r *http.Request) (PageMetadata, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return PageMetadata{
			Offset: DefOffset,
			Limit:  DefLimit,
			Order:  IDOrder,
			Dir:    DescDir,
		}, nil
	}

	var pm PageMetadata
	if err := json.NewDecoder(r.Body).Decode(&pm); err != nil {
		return PageMetadata{}, errors.Wrap(ErrMalformedEntity, err)
	}

	if pm.Limit == 0 {
		pm.Limit = DefLimit
	}

	if pm.Offset == 0 {
		pm.Offset = DefOffset
	}

	if pm.Order == "" {
		pm.Order = IDOrder
	}

	if pm.Dir == "" {
		pm.Dir = DescDir
	}

	return pm, nil
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
