//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/re"

// {"sql":"create stream my_stream (id bigint, name string, score float) WITH ( topic = \"topic/temperature\", FORMAT = \"json\", KEY = \"id\")"}
type streamReq struct {
	token string
	Name  string `json:"name,omitempty"`
	Row   string `json:"row"`
	Topic string `json:"topic"`
}

func (req streamReq) validate() error {
	if req.token == "" {
		return re.ErrMalformedEntity
	}
	if req.Name == "" {
		return re.ErrMalformedEntity
	}
	if req.Row == "" {
		return re.ErrMalformedEntity
	}
	if req.Topic == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type getReq struct {
	token string
}

func (req getReq) validate() error {
	if req.token == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type viewStreamReq struct {
	token string
	name  string
}

func (req viewStreamReq) validate() error {
	if req.token == "" {
		return re.ErrMalformedEntity
	}
	if req.name == "" {
		return re.ErrMalformedEntity
	}
	return nil
}
