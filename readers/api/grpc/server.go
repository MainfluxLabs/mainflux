// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	jsonmsg "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	senmlmsg "github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ protomfx.ReadersServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	listJSONMessages  kitgrpc.Handler
	listSenMLMessages kitgrpc.Handler
}

// NewServer returns new ReadersServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc readers.Service) protomfx.ReadersServiceServer {
	return &grpcServer{
		listJSONMessages: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "list_json_messages")(listJSONMessagesEndpoint(svc)),
			decodeListJSONMessagesRequest,
			encodeListJSONMessagesResponse,
		),
		listSenMLMessages: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "list_senml_messages")(listSenMLMessagesEndpoint(svc)),
			decodeListSenMLMessagesRequest,
			encodeListSenMLMessagesResponse,
		),
	}
}

func (gs *grpcServer) ListJSONMessages(ctx context.Context, req *protomfx.ListJSONMessagesReq) (*protomfx.ListJSONMessagesRes, error) {
	_, res, err := gs.listJSONMessages.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.ListJSONMessagesRes), nil
}

func (gs *grpcServer) ListSenMLMessages(ctx context.Context, req *protomfx.ListSenMLMessagesReq) (*protomfx.ListSenMLMessagesRes, error) {
	_, res, err := gs.listSenMLMessages.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*protomfx.ListSenMLMessagesRes), nil
}

func decodeListJSONMessagesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ListJSONMessagesReq)
	return listJSONMessagesReq{
		thingKey: domain.ThingKey{Value: req.GetThingKey().GetValue(), Type: req.GetThingKey().GetType()},
		pm: domain.JSONPageMetadata{
			Offset:      req.GetOffset(),
			Limit:       req.GetLimit(),
			Subtopic:    req.GetSubtopic(),
			Publisher:   req.GetPublisher(),
			Protocol:    req.GetProtocol(),
			From:        req.GetFrom(),
			To:          req.GetTo(),
			Filter:      req.GetFilter(),
			AggInterval: req.GetAggInterval(),
			AggValue:    req.GetAggValue(),
			AggType:     req.GetAggType(),
			AggFields:   req.GetAggFields(),
			Dir:         req.GetDir(),
		},
	}, nil
}

func decodeListSenMLMessagesRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*protomfx.ListSenMLMessagesReq)
	return listSenMLMessagesReq{
		thingKey: domain.ThingKey{Value: req.GetThingKey().GetValue(), Type: req.GetThingKey().GetType()},
		pm: domain.SenMLPageMetadata{
			Offset:      req.GetOffset(),
			Limit:       req.GetLimit(),
			Subtopic:    req.GetSubtopic(),
			Publisher:   req.GetPublisher(),
			Protocol:    req.GetProtocol(),
			Name:        req.GetName(),
			Value:       req.GetValue(),
			Comparator:  req.GetComparator(),
			BoolValue:   req.GetBoolValue(),
			StringValue: req.GetStringValue(),
			DataValue:   req.GetDataValue(),
			From:        req.GetFrom(),
			To:          req.GetTo(),
			AggInterval: req.GetAggInterval(),
			AggValue:    req.GetAggValue(),
			AggType:     req.GetAggType(),
			AggFields:   req.GetAggFields(),
			Dir:         req.GetDir(),
		},
	}, nil
}

func encodeListJSONMessagesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(listJSONMessagesRes)
	msgs := make([]*protomfx.Message, 0, len(res.page.Messages))
	for _, m := range res.page.Messages {
		jm, ok := m.(jsonmsg.Message)
		if !ok {
			continue
		}
		pm := jm.ToProtoMessage()
		msgs = append(msgs, &pm)
	}
	return &protomfx.ListJSONMessagesRes{
		Total:    res.page.Total,
		Messages: msgs,
	}, nil
}

func encodeListSenMLMessagesResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(listSenMLMessagesRes)
	msgs := make([]*protomfx.Message, 0, len(res.page.Messages))
	for _, m := range res.page.Messages {
		sm, ok := m.(senmlmsg.Message)
		if !ok {
			continue
		}
		payload, err := json.Marshal(sm)
		if err != nil {
			continue
		}
		msgs = append(msgs, &protomfx.Message{
			Publisher:   sm.Publisher,
			Subtopic:    sm.Subtopic,
			Protocol:    sm.Protocol,
			Payload:     payload,
			ContentType: "application/senml+json",
		})
	}
	return &protomfx.ListSenMLMessagesRes{
		Total:    res.page.Total,
		Messages: msgs,
	}, nil
}

func encodeError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case err == nil:
		return nil
	case errors.Contains(err, errors.ErrAuthentication):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, dbutil.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
