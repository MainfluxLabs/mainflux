//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/re"

type streamReq struct {
	token  string
	stream re.Stream
}

func (req streamReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	if req.stream.Name == "" {
		return re.ErrMalformedEntity
	}
	if req.stream.Row == "" {
		return re.ErrMalformedEntity
	}
	if req.stream.Topic == "" {
		return re.ErrMalformedEntity
	}
	if req.stream.Host == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type getReq struct {
	token string
}

func (req getReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	return nil
}

type viewReq struct {
	token string
	name  string
}

func (req viewReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return re.ErrMalformedEntity
	}
	return nil
}

type ruleReq struct {
	token string
	name  string
	Rule  re.Rule
}

func (req ruleReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	return nil
}

type controlReq struct {
	token  string
	name   string
	action string
}

func (req controlReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return re.ErrMalformedEntity
	}
	if !(req.action == "start" || req.action == "stop" || req.action == "restart") {
		return re.ErrMalformedEntity
	}
	return nil
}

type deleteReq struct {
	token string
	name  string
	kind  string
}

func (req deleteReq) validate() error {
	if req.token == "" {
		return re.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return re.ErrMalformedEntity
	}
	if !(req.kind == "streams" || req.kind == "rules") {
		return re.ErrMalformedEntity
	}
	return nil
}
