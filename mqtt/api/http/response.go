// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package http

import "github.com/MainfluxLabs/mainflux/pkg/apiutil"

var _ apiutil.Response = (*listSubscriptionsRes)(nil)

type listSubscriptionsRes struct {
	pageRes
	Subscriptions []viewSubRes `json:"subscriptions,omitempty"`
}

func (res listSubscriptionsRes) Code() int {
	return 200
}

func (res listSubscriptionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listSubscriptionsRes) Empty() bool {
	return false
}

type viewSubRes struct {
	Subtopic  string  `json:"subtopic"`
	ThingID   string  `json:"thing_id"`
	ChannelID string  `json:"channel_id"`
	ClientID  string  `json:"client_id"`
	Status    string  `json:"status"`
	CreatedAt float64 `json:"created_at"`
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}
