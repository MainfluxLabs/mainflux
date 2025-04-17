package mocks

import (
	"context"
	"sync"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type rulesServiceMock struct {
	mu sync.Mutex
}

func NewRulesServiceClient() protomfx.RulesServiceClient {
	return &rulesServiceMock{}
}

func (r rulesServiceMock) Publish(ctx context.Context, in *protomfx.PublishReq, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
