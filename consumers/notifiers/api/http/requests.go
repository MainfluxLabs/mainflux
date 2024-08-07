// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"regexp"
	"strings"

	"github.com/MainfluxLabs/mainflux/internal/email"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// phoneRegexp represent regex pattern to validate E.164 phone numbers
var phoneRegexp = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type apiReq interface {
	validate() error
}

type notifierReq struct {
	token string
	id    string
}

func (req *notifierReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type createNotifierReq struct {
	Contacts []string `json:"contacts"`
}
type createNotifiersReq struct {
	token     string
	groupID   string
	Notifiers []createNotifierReq `json:"notifiers"`
}

func (req createNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.Notifiers) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, nf := range req.Notifiers {
		if err := nf.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (req createNotifierReq) validate() error {
	for _, c := range req.Contacts {
		if !email.IsEmail(c) && !isPhoneNumber(c) {
			return errors.ErrMalformedEntity
		}
	}

	return nil
}

type updateNotifierReq struct {
	token    string
	id       string
	Contacts []string `json:"contacts"`
}

func (req updateNotifierReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.Contacts == nil {
		return errors.ErrMalformedEntity
	}

	for _, c := range req.Contacts {
		if !email.IsEmail(c) && !isPhoneNumber(c) {
			return errors.ErrMalformedEntity
		}
	}

	return nil
}

type removeNotifiersReq struct {
	groupID     string
	token       string
	NotifierIDs []string `json:"notifier_ids,omitempty"`
}

func (req removeNotifiersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.NotifierIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, nfID := range req.NotifierIDs {
		if nfID == "" {
			return apiutil.ErrMissingID
		}
	}

	return nil
}

func isPhoneNumber(phoneNumber string) bool {
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

	return phoneRegexp.MatchString(phoneNumber)
}
