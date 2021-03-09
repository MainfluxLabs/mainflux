// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/mainflux/mainflux/rules"

type streamReq struct {
	token  string
	stream rules.Stream
}

func (req streamReq) validate() error {
	if req.token == "" {
		return rules.ErrUnauthorizedAccess
	}
	if req.stream.Name == "" {
		return rules.ErrMalformedEntity
	}
	if req.stream.Row == "" {
		return rules.ErrMalformedEntity
	}
	if req.stream.Topic == "" {
		return rules.ErrMalformedEntity
	}
	if req.stream.Host == "" {
		return rules.ErrMalformedEntity
	}
	return nil
}

type listReq struct {
	token string
}

func (req listReq) validate() error {
	if req.token == "" {
		return rules.ErrUnauthorizedAccess
	}
	return nil
}

type viewReq struct {
	token string
	name  string
}

func (req viewReq) validate() error {
	if req.token == "" {
		return rules.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return rules.ErrMalformedEntity
	}
	return nil
}

type ruleReq struct {
	token          string
	ID             string `json:"id"`
	Sql            string `json:"sql"`
	Host           string `json:"host"`
	Port           string `json:"port"`
	Channel        string `json:"channel"`
	Subtopic       string `json:"subtopic"`
	SendToMetasink bool
}

func (req ruleReq) validate() error {
	if req.token == "" {
		return rules.ErrUnauthorizedAccess
	}
	if req.Host == "" {
		return rules.ErrMalformedEntity
	}
	if req.Channel == "" {
		return rules.ErrMalformedEntity
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
		return rules.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return rules.ErrMalformedEntity
	}
	if req.action != "start" && req.action != "stop" && req.action != "restart" {
		return rules.ErrMalformedEntity
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
		return rules.ErrUnauthorizedAccess
	}
	if req.name == "" {
		return rules.ErrMalformedEntity
	}
	if !(req.kind == "streams" || req.kind == "rules") {
		return rules.ErrMalformedEntity
	}
	return nil
}
