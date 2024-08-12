// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

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

func SendRequest(fullURL, method string, body interface{}, headers map[string]string) ([]byte, error) {
	_, err := url.ParseRequestURI(fullURL)
	if err != nil {
		return nil, err
	}

	var jsonData []byte
	if body != nil {
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewReader(jsonData))
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

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return responseData, nil
}
