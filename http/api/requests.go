// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type publishReq struct {
	msg protomfx.Message
	apiutil.ThingKey
}

func (req publishReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
	}

	return nil
}
