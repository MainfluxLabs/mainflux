// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"html/template"
	"net/http"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*uiRes)(nil)
)

type uiRes struct {
	template *template.Template
	name     string
	data     interface{}
}

func (res uiRes) Code() int {
	return http.StatusCreated
}

func (res uiRes) Headers() map[string]string {
	return map[string]string{}
}

func (res uiRes) Empty() bool {
	return false
}
