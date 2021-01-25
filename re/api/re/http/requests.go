//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/re"

type apiReq interface {
	validate() error
}

type pingReq struct {
	Secret string
}

func (req pingReq) validate() error {
	if req.Secret == "" {
		return re.ErrUnauthorizedAccess
	}

	return nil
}

// {"sql":"create stream my_stream (id bigint, name string, score float) WITH ( datasource = \"topic/temperature\", FORMAT = \"json\", KEY = \"id\")"}
type streamReq struct {
	SQL string `json:"sql"`
}

func (req streamReq) validate() error {
	if req.SQL == "" {
		return re.ErrMalformedEntity
	}
	return nil
}
