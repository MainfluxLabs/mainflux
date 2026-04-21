// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

const maxLimitSize = 200

type addCertsReq struct {
	token   string
	ThingID string `json:"thing_id"`
	KeyBits int    `json:"key_bits"`
	KeyType string `json:"key_type"`
	TTL     string `json:"ttl"`
}

func (req addCertsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ThingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.TTL == "" {
		return apiutil.ErrMissingCertData
	}

	if err := validateKeyParams(req.KeyType, req.KeyBits); err != nil {
		return err
	}

	return nil
}

type rotateCertReq struct {
	serial  string
	token   string
	ThingID string `json:"thing_id"`
	KeyBits int    `json:"key_bits"`
	KeyType string `json:"key_type"`
	TTL     string `json:"ttl"`
}

func (req rotateCertReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.serial == "" {
		return apiutil.ErrMissingSerial
	}

	if req.ThingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.TTL == "" {
		return apiutil.ErrMissingCertData
	}

	if err := validateKeyParams(req.KeyType, req.KeyBits); err != nil {
		return err
	}

	return nil
}

type listReq struct {
	thingID string
	token   string
	offset  uint64
	limit   uint64
}

func (req *listReq) validate() error {
	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type viewReq struct {
	serial string
	token  string
}

func (req *viewReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.serial == "" {
		return apiutil.ErrMissingSerial
	}

	return nil
}

type downloadReq struct {
	serial   string
	thingKey domain.ThingKey
}

func (req *downloadReq) validate() error {
	if req.serial == "" {
		return apiutil.ErrMissingSerial
	}

	if req.thingKey.Value == "" {
		return apiutil.ErrMissingAuth
	}

	return apiutil.ValidateThingKey(req.thingKey)
}

type revokeReq struct {
	token  string
	serial string
}

func (req *revokeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.serial == "" {
		return apiutil.ErrMissingSerial
	}

	return nil
}

func validateKeyParams(keyType string, keyBits int) error {
	switch keyType {
	case pki.RSAKeyType:
		if keyBits != pki.RSAKeyBits2048 && keyBits != pki.RSAKeyBits4096 {
			return apiutil.ErrMissingCertData
		}
	case pki.ECDSAKeyType:
		if keyBits != pki.ECDSAKeyBits224 && keyBits != pki.ECDSAKeyBits256 && keyBits != pki.ECDSAKeyBits384 && keyBits != pki.ECDSAKeyBits521 {
			return apiutil.ErrMissingCertData
		}
	default:
		return apiutil.ErrMissingCertData
	}
	return nil
}
