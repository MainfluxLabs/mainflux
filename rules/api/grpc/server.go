package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/rules"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ protomfx.RulesServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	performActions kitgrpc.Handler
}

func NewServer(tracer opentracing.Tracer, svc rules.Service) protomfx.RulesServiceServer {
	return &grpcServer{
		performActions: kitgrpc.NewServer(
			kitot.TraceServer(tracer, "publish")(publishEndpoint(svc)),
			decodePublishRequest,
			encodeEmptyResponse,
		),
	}
}

func (gs grpcServer) Publish(ctx context.Context, message *protomfx.PublishReq) (*emptypb.Empty, error) {
	_, res, err := gs.performActions.ServeGRPC(ctx, message)
	if err != nil {
		return nil, nil
	}

	return res.(*emptypb.Empty), nil
}

func decodePublishRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*protomfx.PublishReq)
	return publishReq{message: *req.Message}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &emptypb.Empty{}, encodeError(res.err)
}

func encodeError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case err == nil:
		return nil
	case errors.Contains(err, apiutil.ErrUnsupportedContentType),
		err == ErrMissingPayload:
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
