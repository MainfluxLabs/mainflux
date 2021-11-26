// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*uiRes)(nil)
)

type uiRes struct {
	code    int
	headers map[string]string
	html    []byte
}

func (res uiRes) Code() int {
	if res.code == 0 {
		return http.StatusCreated
	}

	return res.code
}

func (res uiRes) Headers() map[string]string {
	if res.headers == nil {
		return map[string]string{}
	}

	return res.headers
}

func (res uiRes) Empty() bool {
	return res.html == nil
}
