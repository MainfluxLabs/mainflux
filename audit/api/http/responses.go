// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ apiutil.Response = (*eventsPageRes)(nil)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

type eventRes struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Operation  string          `json:"operation"`
	Actor      domain.Identity `json:"actor"`
	OrgID      string          `json:"org_id,omitempty"`
	GroupID    string          `json:"group_id,omitempty"`
	ActionData map[string]any  `json:"action_data,omitempty"`
}

type eventsPageRes struct {
	pageRes
	Events []eventRes `json:"events"`
}

func (res eventsPageRes) Code() int {
	return http.StatusOK
}

func (res eventsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res eventsPageRes) Empty() bool {
	return false
}
