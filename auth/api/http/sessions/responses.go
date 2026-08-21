// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sessions

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*logoutRes)(nil)
)

type logoutRes struct{}

func (res logoutRes) Code() int {
	return http.StatusNoContent
}

func (res logoutRes) Headers() map[string]string {
	return map[string]string{}
}

func (res logoutRes) Empty() bool {
	return true
}
