// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
)

type webhookResponse struct {
	ID             string `json:"id"`
	GroupID        string `json:"group_id"`
	Name           string `json:"name"`
	Url            string `json:"url"`
	WebhookHeaders string `json:"headers"`
}

type webhooksRes struct {
	Webhooks []webhookResponse `json:"webhooks"`
	created  bool
}

func (res webhooksRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res webhooksRes) Headers() map[string]string {
	return map[string]string{}
}

func (res webhooksRes) Empty() bool {
	return false
}
