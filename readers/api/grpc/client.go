// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	jsonmsg "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	senmlmsg "github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ domain.ReadersClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout           time.Duration
	listJSONMessages  endpoint.Endpoint
	listSenMLMessages endpoint.Endpoint
}

// NewClient returns new gRPC client instance implementing domain.ReadersClient.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) domain.ReadersClient {
	svcName := "protomfx.ReadersService"

	return &grpcClient{
		timeout: timeout,
		listJSONMessages: kitot.TraceClient(tracer, "list_json_messages")(kitgrpc.NewClient(
			conn,
			svcName,
			"ListJSONMessages",
			encodeListJSONMessagesRequest,
			decodeListJSONMessagesResponse,
			protomfx.ListJSONMessagesRes{},
		).Endpoint()),
		listSenMLMessages: kitot.TraceClient(tracer, "list_senml_messages")(kitgrpc.NewClient(
			conn,
			svcName,
			"ListSenMLMessages",
			encodeListSenMLMessagesRequest,
			decodeListSenMLMessagesResponse,
			protomfx.ListSenMLMessagesRes{},
		).Endpoint()),
	}
}

func (c grpcClient) ListJSONMessages(ctx context.Context, key domain.ThingKey, pm domain.JSONPageMetadata) (domain.JSONMessagesPage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.listJSONMessages(ctx, listJSONMessagesReq{thingKey: key, pm: pm})
	if err != nil {
		return domain.JSONMessagesPage{}, err
	}

	r := res.(listJSONMessagesRes)
	return r.page, nil
}

func (c grpcClient) ListSenMLMessages(ctx context.Context, key domain.ThingKey, pm domain.SenMLPageMetadata) (domain.SenMLMessagesPage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	res, err := c.listSenMLMessages(ctx, listSenMLMessagesReq{thingKey: key, pm: pm})
	if err != nil {
		return domain.SenMLMessagesPage{}, err
	}

	r := res.(listSenMLMessagesRes)
	return r.page, nil
}

func encodeListJSONMessagesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(listJSONMessagesReq)
	return &protomfx.ListJSONMessagesReq{
		ThingKey:    &protomfx.ThingKey{Value: req.thingKey.Value, Type: req.thingKey.Type},
		Offset:      req.pm.Offset,
		Limit:       req.pm.Limit,
		Subtopic:    req.pm.Subtopic,
		Publisher:   req.pm.Publisher,
		Protocol:    req.pm.Protocol,
		From:        req.pm.From,
		To:          req.pm.To,
		Filter:      req.pm.Filter,
		AggInterval: req.pm.AggInterval,
		AggValue:    req.pm.AggValue,
		AggType:     req.pm.AggType,
		AggFields:   req.pm.AggFields,
		Dir:         req.pm.Dir,
	}, nil
}

func encodeListSenMLMessagesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(listSenMLMessagesReq)
	return &protomfx.ListSenMLMessagesReq{
		ThingKey:    &protomfx.ThingKey{Value: req.thingKey.Value, Type: req.thingKey.Type},
		Offset:      req.pm.Offset,
		Limit:       req.pm.Limit,
		Subtopic:    req.pm.Subtopic,
		Publisher:   req.pm.Publisher,
		Protocol:    req.pm.Protocol,
		Name:        req.pm.Name,
		Value:       req.pm.Value,
		Comparator:  req.pm.Comparator,
		BoolValue:   req.pm.BoolValue,
		StringValue: req.pm.StringValue,
		DataValue:   req.pm.DataValue,
		From:        req.pm.From,
		To:          req.pm.To,
		AggInterval: req.pm.AggInterval,
		AggValue:    req.pm.AggValue,
		AggType:     req.pm.AggType,
		AggFields:   req.pm.AggFields,
		Dir:         req.pm.Dir,
	}, nil
}

func decodeListJSONMessagesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.ListJSONMessagesRes)
	msgs := make([]domain.Message, 0, len(res.GetMessages()))
	for _, pm := range res.GetMessages() {
		msgs = append(msgs, jsonmsg.Message{
			Created:   pm.GetCreated(),
			Subtopic:  pm.GetSubtopic(),
			Publisher: pm.GetPublisher(),
			Protocol:  pm.GetProtocol(),
			Payload:   pm.GetPayload(),
		})
	}

	return listJSONMessagesRes{
		page: domain.JSONMessagesPage{
			MessagesPage: domain.MessagesPage{
				Total:    res.GetTotal(),
				Messages: msgs,
			},
		},
	}, nil
}

func decodeListSenMLMessagesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*protomfx.ListSenMLMessagesRes)

	msgs := make([]domain.Message, 0, len(res.GetMessages()))
	for _, pm := range res.GetMessages() {
		var sm senmlmsg.Message
		if err := json.Unmarshal(pm.GetPayload(), &sm); err != nil {
			continue
		}
		msgs = append(msgs, sm)
	}

	return listSenMLMessagesRes{
		page: domain.SenMLMessagesPage{
			MessagesPage: domain.MessagesPage{
				Total:    res.GetTotal(),
				Messages: msgs,
			},
		},
	}, nil
}
