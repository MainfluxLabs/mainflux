// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	//"errors"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Ping compares a given string with secret
	CreateWebhook(string) (bool, error)
}

type webhooksService struct {
	secret string
}

var _ Service = (*webhooksService)(nil)

// New instantiates the webhooks service implementation.
func New(secret string) Service {
	return &webhooksService{
		secret: secret,
	}
}

func (ks *webhooksService) CreateWebhook(secret string) (bool, error) {
	if ks.secret != secret {
		return false, ErrUnauthorizedAccess
	}
	return true, nil
}
