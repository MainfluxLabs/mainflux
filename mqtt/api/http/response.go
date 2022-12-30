// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package http

import "github.com/MainfluxLabs/mainflux"

var _ mainflux.Response = (*listAllSubscriptionsRes)(nil)

type listAllSubscriptionsRes struct {
	pageRes
	Subscriptions []viewSubRes `json:"subscriptions,omitempty"`
}

func (res listAllSubscriptionsRes) Code() int {
	return 200
}

func (res listAllSubscriptionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listAllSubscriptionsRes) Empty() bool {
	return false
}

type viewSubRes struct {
	ID       string `json:"id"`
	OwnerID  string `json:"owner_id"`
	Subtopic string `json:"subtopic"`
	ThingID  string `json:"thing_id"`
	ChanID   string `json:"chan_id"`
}

type pageRes struct {
	Total     uint64 `json:"total"`
	Offset    uint64 `json:"offset"`
	Limit     uint64 `json:"limit"`
	Order     string `json:"order"`
	Direction string `json:"direction"`
}
