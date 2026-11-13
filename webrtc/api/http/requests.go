// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webrtc"
)

type iceServersReq struct {
	token string
}

func (req iceServersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type viewReq struct {
	token   string
	thingID string
	SDP     string `json:"sdp"`
}

func (req viewReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.SDP == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type addCandidateReq struct {
	token     string
	sessionID string
	Candidate webrtc.ICECandidate `json:"candidate"`
}

func (req addCandidateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.sessionID == "" || req.Candidate.Candidate == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}
