// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

type publishReq struct {
	msg protomfx.Message
	things.ThingKey
}

func (req publishReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
	}

	return nil
}
