// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

// Response contains HTTP response specific methods.
type Response interface {
	// Code returns HTTP response code.
	Code() int

	// Headers returns map of HTTP headers with their values.
	Headers() map[string]string

	// Empty indicates if HTTP response has content.
	Empty() bool
}

// ErrorRes represents the HTTP error response body.
type ErrorRes struct {
	Err string `json:"error"`
}
