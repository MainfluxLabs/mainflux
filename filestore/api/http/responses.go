// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ apiutil.Response = (*fileRes)(nil)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	Name   string `json:"name,omitempty"`
}

type fileInfo struct {
	Name     string         `json:"name"`
	Class    string         `json:"class"`
	Format   string         `json:"format"`
	Time     float64        `json:"time"`
	Metadata map[string]any `json:"metadata"`
}

type fileRes struct {
	created bool
}

func (res fileRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res fileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res fileRes) Empty() bool {
	return false
}

type listFilesRes struct {
	pageRes
	FilesInfo []fileInfo `json:"files_info,omitempty"`
}

func (res listFilesRes) Code() int {
	return http.StatusOK
}

func (res listFilesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listFilesRes) Empty() bool {
	return false
}

type viewFileRes struct {
	file []byte
}

func (res viewFileRes) Code() int {
	return http.StatusOK
}

func (res viewFileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewFileRes) Empty() bool {
	return false
}

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}
