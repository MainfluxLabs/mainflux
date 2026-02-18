// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	contentType   = "Content-Type"
	ctJSON        = "application/json"
	defaultDelay  = 1 * time.Second
	maxRetryDelay = 10 * time.Second
	maxRetries    = 3
)

var (
	httpClient        = &http.Client{Timeout: 30 * time.Second}
	errSendRequest    = errors.New("failed to send request")
	retryDelayHeaders = []string{"Retry-After", "RateLimit-Reset", "X-RateLimit-Reset", "X-Rate-Limit-Reset"}
)

func SendRequest(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	var resErr error

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest(method, path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		if req.Header.Get(contentType) == "" {
			req.Header.Set(contentType, ctJSON)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			if !isRetryableError(err) {
				return nil, errors.Wrap(errSendRequest, err)
			}
			resErr = err
			time.Sleep(defaultDelay)
			continue
		}

		if isRetryableStatus(resp.StatusCode) {
			delay := getRetryDelay(resp)
			resp.Body.Close()

			if i < maxRetries-1 {
				time.Sleep(delay)
			}
			continue
		}

		return resp, nil
	}

	return nil, errors.Wrap(errSendRequest, resErr)
}

func isRetryableError(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooEarly ||
		status == http.StatusTooManyRequests ||
		(status >= http.StatusInternalServerError && status != http.StatusNotImplemented)
}

func getRetryDelay(resp *http.Response) time.Duration {
	if resp == nil {
		return defaultDelay
	}

	for _, header := range retryDelayHeaders {
		val := resp.Header.Get(header)
		if val == "" {
			continue
		}

		seconds, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}

		if seconds > 1e9 {
			delay := time.Duration(seconds-time.Now().Unix()) * time.Second
			if delay > 0 && delay <= maxRetryDelay {
				return delay
			}
			continue
		}

		delay := time.Duration(seconds) * time.Second
		if delay <= maxRetryDelay {
			return delay
		}
	}

	return defaultDelay
}
