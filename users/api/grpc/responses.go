// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"

type getUsersRes struct {
	users  []*protomfx.User
	total  uint64
	limit  uint64
	offset uint64
}
