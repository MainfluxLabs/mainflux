package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var ErrMissingPayload = errors.New("missing payload")

type publishReq struct {
	message protomfx.Message
}

func (req publishReq) validate() error {
	if req.message.ContentType == "" {
		return apiutil.ErrUnsupportedContentType
	}

	if len(req.message.Payload) == 0 {
		return ErrMissingPayload
	}

	return nil
}
