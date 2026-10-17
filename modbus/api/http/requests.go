// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	minLen       = 1
	maxStringLen = 125
	maxLimitSize = 100
	maxNameSize  = 254
)

var (
	ErrMissingID           = errors.New("missing client id")
	ErrInvalidScheduler    = errors.New("missing or invalid scheduler")
	ErrMissingIPAddress    = errors.New("missing IP address")
	ErrMissingPort         = errors.New("missing port")
	ErrMissingDataFields   = errors.New("missing data fields")
	ErrInvalidFunctionCode = errors.New("invalid function code")
	ErrMissingFieldName    = errors.New("missing field name")
	ErrMissingFieldAddress = errors.New("missing field address")
	ErrInvalidFieldType    = errors.New("invalid field type")
	ErrInvalidFieldLength  = errors.New("invalid field length")
	ErrInvalidByteOrder    = errors.New("invalid byte order")
	ErrMissingValue        = errors.New("missing value")
	ErrInvalidValue        = errors.New("invalid value")
	ErrValueOutOfRange     = errors.New("value out of range for type")
)

// validatePageMetadata validates the modbus page metadata.
func validatePageMetadata(pm modbus.PageMetadata) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, modbus.ClientOrderFields); err != nil {
		return err
	}

	if len(pm.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type field struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Unit      string  `json:"unit,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	ByteOrder string  `json:"byte_order"`
	Address   *uint16 `json:"address"`
	Length    uint16  `json:"length,omitempty"`
	Value     any     `json:"value,omitempty"`
}

type client struct {
	Name         string         `json:"name"`
	IPAddress    string         `json:"ip_address"`
	Port         string         `json:"port"`
	SlaveID      uint8          `json:"slave_id,omitempty"`
	FunctionCode string         `json:"function_code"`
	Scheduler    cron.Scheduler `json:"scheduler"`
	DataFields   []field        `json:"data_fields"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type createClientsReq struct {
	token   string
	thingID string
	Clients []client `json:"clients"`
}

func (req createClientsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.Clients) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, cl := range req.Clients {
		if err := cl.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req client) validate() error {
	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	if req.IPAddress == "" {
		return ErrMissingIPAddress
	}

	if req.Port == "" {
		return ErrMissingPort
	}

	if !req.Scheduler.IsValid() {
		return ErrInvalidScheduler
	}

	switch req.FunctionCode {
	case modbus.WriteCoilsFunc,
		modbus.WriteHoldingRegistersFunc:
		if err := validateDataFields(req.DataFields); err != nil {
			return err
		}

		return validateWriteFields(req.FunctionCode, req.DataFields)
	default:
		if err := validateFunctionCode(req.FunctionCode); err != nil {
			return err
		}

		return validateDataFields(req.DataFields)
	}
}

// validateWriteFields checks the constraints specific to the data fields of a write client.
func validateWriteFields(funcCode string, fields []field) error {
	for _, f := range fields {
		if funcCode == modbus.WriteCoilsFunc && f.Type != modbus.BoolType {
			return ErrInvalidFieldType
		}

		if err := validateWriteType(f.Type); err != nil {
			return err
		}

		if err := validateWriteValue(f); err != nil {
			return err
		}
	}

	return nil
}

// validateWriteValue checks that the value is a bool for the bool type, and a number within the type range for numeric types.
func validateWriteValue(f field) error {
	if f.Value == nil {
		return ErrMissingValue
	}

	switch f.Type {
	case modbus.BoolType:
		if _, ok := f.Value.(bool); !ok {
			return ErrInvalidValue
		}

		return nil
	default:
		v, ok := f.Value.(float64)
		if !ok {
			return ErrInvalidValue
		}

		if err := modbus.ValidateWriteValue(v, f.Type, f.Scale); err != nil {
			return ErrValueOutOfRange
		}

		return nil
	}
}

// validateFunctionCode checks that code is one of the supported read function codes.
func validateFunctionCode(code string) error {
	switch code {
	case modbus.ReadCoilsFunc, modbus.ReadDiscreteInputsFunc, modbus.ReadInputRegistersFunc, modbus.ReadHoldingRegistersFunc:
		return nil
	default:
		return ErrInvalidFunctionCode
	}
}

// validateByteOrder checks order against the supported byte orders; empty is allowed (use default).
func validateByteOrder(order string) error {
	if order == "" {
		return nil
	}

	switch order {
	case modbus.ByteOrderABCD, modbus.ByteOrderDCBA, modbus.ByteOrderCDAB, modbus.ByteOrderBADC:
		return nil
	default:
		return ErrInvalidByteOrder
	}
}

// validateDataFields checks a non-empty list of data fields.
func validateDataFields(fields []field) error {
	if len(fields) < minLen {
		return ErrMissingDataFields
	}

	for _, f := range fields {
		if f.Name == "" {
			return ErrMissingFieldName
		}

		if f.Address == nil {
			return ErrMissingFieldAddress
		}

		if err := validateFieldType(f.Type, f.Length); err != nil {
			return err
		}

		if err := validateByteOrder(f.ByteOrder); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldType checks t against the supported field types; string fields require a length within the Modbus read limit.
func validateFieldType(t string, length uint16) error {
	switch t {
	case modbus.BoolType, modbus.Int16Type, modbus.Uint16Type, modbus.Int32Type, modbus.Uint32Type, modbus.Float32Type:
		return nil
	case modbus.StringType:
		if length < minLen || length > maxStringLen {
			return ErrInvalidFieldLength
		}
		return nil
	default:
		return ErrInvalidFieldType
	}
}

// validateWriteType checks t against the types writable via a single register or coil write.
func validateWriteType(t string) error {
	switch t {
	case modbus.BoolType, modbus.Int16Type, modbus.Uint16Type, modbus.Int32Type, modbus.Uint32Type, modbus.Float32Type:
		return nil
	default:
		return ErrInvalidFieldType
	}
}

type listClientsByThingReq struct {
	token        string
	thingID      string
	pageMetadata modbus.PageMetadata
}

func (req listClientsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return validatePageMetadata(req.pageMetadata)
}

type listClientsByGroupReq struct {
	token        string
	groupID      string
	pageMetadata modbus.PageMetadata
}

func (req listClientsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return validatePageMetadata(req.pageMetadata)
}

type viewClientReq struct {
	token string
	id    string
}

func (req viewClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return ErrMissingID
	}

	return nil
}

type readRegisterReq struct {
	token        string
	IPAddress    string  `json:"ip_address"`
	Port         string  `json:"port"`
	SlaveID      uint8   `json:"slave_id,omitempty"`
	FunctionCode string  `json:"function_code"`
	Address      *uint16 `json:"address"`
	Type         string  `json:"type,omitempty"`
	ByteOrder    string  `json:"byte_order,omitempty"`
	Scale        float64 `json:"scale,omitempty"`
	Length       uint16  `json:"length,omitempty"`
}

func (req readRegisterReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.IPAddress == "" {
		return ErrMissingIPAddress
	}

	if req.Port == "" {
		return ErrMissingPort
	}

	if err := validateFunctionCode(req.FunctionCode); err != nil {
		return err
	}

	if req.Address == nil {
		return ErrMissingFieldAddress
	}

	switch req.FunctionCode {
	case modbus.ReadCoilsFunc, modbus.ReadDiscreteInputsFunc:
		// Coils and discrete inputs are single bits and take no decoding options.
		return nil
	default:
		if err := validateFieldType(req.Type, req.Length); err != nil {
			return err
		}

		return validateByteOrder(req.ByteOrder)
	}
}

type writeRegisterReq struct {
	token     string
	IPAddress string  `json:"ip_address"`
	Port      string  `json:"port"`
	SlaveID   uint8   `json:"slave_id,omitempty"`
	Address   *uint16 `json:"address"`
	Type      string  `json:"type"`
	ByteOrder string  `json:"byte_order,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	Value     any     `json:"value"`
}

func (req writeRegisterReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.IPAddress == "" {
		return ErrMissingIPAddress
	}

	if req.Port == "" {
		return ErrMissingPort
	}

	if req.Address == nil {
		return ErrMissingFieldAddress
	}

	if req.Value == nil {
		return ErrMissingValue
	}

	if err := validateWriteType(req.Type); err != nil {
		return err
	}

	return validateByteOrder(req.ByteOrder)
}

type updateClientReq struct {
	token string
	id    string
	client
}

func (req updateClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return ErrMissingID
	}

	if err := req.client.validate(); err != nil {
		return err
	}

	return nil
}

type createClientReq struct {
	token   string
	thingID string
	client
}

func (req createClientReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return req.client.validate()
}

type removeClientsReq struct {
	token     string
	ClientIDs []string `json:"client_ids,omitempty"`
}

func (req removeClientsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.ClientIDs) < minLen {
		return apiutil.ErrEmptyList
	}

	for _, dlID := range req.ClientIDs {
		if dlID == "" {
			return ErrMissingID
		}
	}

	return nil
}
