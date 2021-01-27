//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/re"

// {"sql":"create stream my_stream (id bigint, name string, score float) WITH ( datasource = \"topic/temperature\", FORMAT = \"json\", KEY = \"id\")"}
type createStreamReq struct {
	SQL string `json:"sql"`
}

func (req createStreamReq) validate() error {
	if req.SQL == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type viewStreamReq struct {
	// token string
	id string
}

func (req viewStreamReq) validate() error {
	if req.id == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type updateStreamReq struct {
	id  string
	SQL string `json:"sql"`
}

func (req updateStreamReq) validate() error {
	if req.SQL == "" {
		return re.ErrMalformedEntity
	}
	if req.id == "" {
		return re.ErrMalformedEntity
	}
	return nil
}
