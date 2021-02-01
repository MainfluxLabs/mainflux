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
	"github.com/mainflux/mainflux/re"
)

func infoEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// TODO: infoEndpoint uses decodeGet and discards request; how to bypass decode func?
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

func createStreamEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateStream(ctx, req.token, req.Name, req.Topic, req.Row, false)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func updateStreamEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateStream(ctx, req.token, req.Name, req.Topic, req.Row, true)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func listStreamsEndpoint(svc re.Service) endpoint.Endpoint {
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

func viewStreamEndpoint(svc re.Service) endpoint.Endpoint {
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

func deleteStreamEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.Delete(ctx, req.token, req.name, "stream")
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func createRuleEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateRule(ctx, req.token, req.Rule, false)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func updateRuleEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ruleReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateRule(ctx, req.token, req.Rule, true)
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func listRulesEndpoint(svc re.Service) endpoint.Endpoint {
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

func viewRuleEndpoint(svc re.Service) endpoint.Endpoint {
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

func deleteRuleEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.Delete(ctx, req.token, req.name, "rule")
		if err != nil {
			return nil, err
		}

		return resultRes{
			Result: result,
		}, nil
	}
}

func getRuleStatusEndpoint(svc re.Service) endpoint.Endpoint {
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
