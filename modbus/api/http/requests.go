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
	ErrInvalidFieldType    = errors.New("invalid field type")
	ErrInvalidFieldLength  = errors.New("invalid field length")
	ErrInvalidByteOrder    = errors.New("invalid byte order")
)

type field struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Unit      string  `json:"unit,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	ByteOrder string  `json:"byte_order"`
	Address   uint16  `json:"address"`
	Length    uint16  `json:"length,omitempty"`
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
	case modbus.ReadCoilsFunc,
		modbus.ReadDiscreteInputsFunc,
		modbus.ReadInputRegistersFunc,
		modbus.ReadHoldingRegistersFunc:
	default:
		return ErrInvalidFunctionCode
	}

	if len(req.DataFields) < minLen {
		return ErrMissingDataFields
	}

	for _, f := range req.DataFields {
		if f.Name == "" {
			return ErrMissingFieldName
		}

		switch f.Type {
		case modbus.BoolType, modbus.Int16Type, modbus.Uint16Type, modbus.Int32Type, modbus.Uint32Type, modbus.Float32Type:
		case modbus.StringType:
			if f.Length < minLen {
				return ErrInvalidFieldLength
			}
		default:
			return ErrInvalidFieldType
		}

		if f.ByteOrder != "" {
			switch f.ByteOrder {
			case modbus.ByteOrderABCD, modbus.ByteOrderDCBA, modbus.ByteOrderCDAB, modbus.ByteOrderBADC:
			default:
				return ErrInvalidByteOrder
			}
		}
	}

	return nil
}

type listClientsByThingReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listClientsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type listClientsByGroupReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listClientsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
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
