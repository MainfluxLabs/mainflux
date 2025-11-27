package grpc

import (
	"context"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.RulesServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout time.Duration
	publish endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, tracer opentracing.Tracer, timeout time.Duration) protomfx.RulesServiceClient {
	svcName := "protomfx.RulesService"
	return &grpcClient{
		timeout: timeout,
		publish: kitot.TraceClient(tracer, "publish")(kitgrpc.NewClient(
			conn,
			svcName,
			"Publish",
			encodePublishRequest,
			decodeEmptyResponse,
			emptypb.Empty{},
		).Endpoint()),
	}
}

func (gc grpcClient) Publish(ctx context.Context, req *protomfx.PublishReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, gc.timeout)
	defer cancel()

	r := publishReq{message: *req.Message}
	res, err := gc.publish(ctx, r)
	if err != nil {
		return nil, err
	}

	er := res.(emptyRes)
	return &emptypb.Empty{}, er.err
}

func encodePublishRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(publishReq)
	return &protomfx.PublishReq{Message: &req.message}, nil
}

func decodeEmptyResponse(_ context.Context, _ interface{}) (interface{}, error) {
	return emptyRes{}, nil
}
