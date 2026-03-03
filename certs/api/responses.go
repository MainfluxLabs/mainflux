// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"time"
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type certsPageRes struct {
	pageRes
	Certs []certsRes `json:"certs"`
}

type certsRes struct {
	ThingID    string    `json:"thing_id"`
	CertSerial string    `json:"cert_serial"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type issueCertRes struct {
	Certificate    string    `json:"certificate"`
	IssuingCA      string    `json:"issuing_ca"`
	CAChain        []string  `json:"ca_chain"`
	PrivateKey     string    `json:"private_key"`
	PrivateKeyType string    `json:"private_key_type"`
	Serial         string    `json:"serial"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type viewCertRes struct {
	Certificate string    `json:"certificate"`
	Serial      string    `json:"serial"`
	ExpiresAt   time.Time `json:"expires_at"`
	ThingID     string    `json:"thing_id"`
}

func (res certsPageRes) Code() int {
	return http.StatusOK
}

func (res certsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsPageRes) Empty() bool {
	return false
}

func (res certsRes) Code() int {
	return http.StatusOK
}

func (res certsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsRes) Empty() bool {
	return false
}

func (res issueCertRes) Code() int {
	return http.StatusCreated
}

func (res issueCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res issueCertRes) Empty() bool {
	return false
}

func (res viewCertRes) Code() int {
	return http.StatusOK
}

func (res viewCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewCertRes) Empty() bool {
	return false
}
