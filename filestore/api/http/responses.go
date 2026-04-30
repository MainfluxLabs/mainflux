// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ apiutil.Response = (*fileRes)(nil)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
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

type streamFileRes struct {
	reader io.ReadCloser
	name   string
}

func (res streamFileRes) Code() int {
	return http.StatusOK
}

func (res streamFileRes) Headers() map[string]string {
	h := map[string]string{}
	if res.name != "" {
		h["Content-Disposition"] = contentDisposition(res.name)
	}
	return h
}

// contentDisposition builds an RFC 6266 attachment header. ASCII names are
// emitted with a quoted filename; non-ASCII names additionally include a
// filename* parameter per RFC 5987 so UTF-8 is conveyed faithfully.
func contentDisposition(name string) string {
	if isASCII(name) {
		return fmt.Sprintf(`attachment; filename=%q`, name)
	}
	ascii := strings.Map(func(r rune) rune {
		if r > 0x7f {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf(`attachment; filename=%q; filename*=UTF-8''%s`, ascii, url.PathEscape(name))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return true
}

func (res streamFileRes) Empty() bool {
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
