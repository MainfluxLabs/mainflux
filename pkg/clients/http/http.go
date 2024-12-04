// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	contentType = "Content-Type"
	ctJSON      = "application/json"
)

var (
	httpClient     = &http.Client{}
	ErrSendRequest = errors.New("failed to send request")
)

func SendRequest(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	if req.Header.Get(contentType) == "" {
		req.Header.Set(contentType, ctJSON)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}
