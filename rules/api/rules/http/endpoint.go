//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/rules"
)

func infoEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		info, err := svc.Info(ctx)
		if err != nil {
			return nil, err
		}
		return infoRes{
			Version:       info.Version,
			Os:            info.Os,
			UpTimeSeconds: info.UpTimeSeconds,
		}, nil
	}
}

func createStreamEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateStream(ctx, req.token, req.stream)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func updateStreamEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.UpdateStream(ctx, req.token, req.stream)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func listStreamsEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		streams, err := svc.ListStreams(ctx, req.token)
		if err != nil {
			return nil, err
		}
		return listStreamsRes{
			Streams: streams,
		}, nil
	}
}

func viewStreamEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		stream, err := svc.ViewStream(ctx, req.token, req.name)
		if err != nil {
			return nil, err
		}
		return viewStreamRes{
			Stream: stream,
		}, nil
	}
}

func deleteEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.Delete(ctx, req.token, req.name, req.kind)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func createRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateRule(ctx, req.token, req.Rule)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func updateRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.UpdateRule(ctx, req.token, req.Rule)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func listRulesEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		rules, err := svc.ListRules(ctx, req.token)
		if err != nil {
			return nil, err
		}
		return listRulesRes{
			Rules: rules,
		}, nil
	}
}

func viewRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		rule, err := svc.ViewRule(ctx, req.token, req.name)
		if err != nil {
			return nil, err
		}
		return viewRuleRes{
			Rule: rule,
		}, nil
	}
}

func getRuleStatusEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var status statusRes
		status, err := svc.GetRuleStatus(ctx, req.token, req.name)
		if err != nil {
			return nil, err
		}

		return status, nil
	}
}

func controlRuleEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(controlReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.ControlRule(ctx, req.token, req.name, req.action)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}
