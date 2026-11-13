// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/webrtc"
	"github.com/go-kit/kit/endpoint"
)

func viewICEServersEndpoint(svc webrtc.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(iceServersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		servers, err := svc.ViewICEServers(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return iceServersRes{ICEServers: servers}, nil
	}
}

func viewEndpoint(svc webrtc.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, err := svc.View(ctx, req.token, req.thingID, req.SDP)
		if err != nil {
			return nil, err
		}

		return viewRes{SessionID: session.ID, SDP: session.SDP}, nil
	}
}

func addCandidateEndpoint(svc webrtc.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(addCandidateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AddViewerCandidate(ctx, req.token, req.sessionID, req.Candidate); err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusNoContent}, nil
	}
}
