//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/re"

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

type viewReq struct {
	// token string
	id string
}

func (req viewReq) validate() error {
	if req.id == "" {
		return re.ErrMalformedEntity
	}
	return nil
}
