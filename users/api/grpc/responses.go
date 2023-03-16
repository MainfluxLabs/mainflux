// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux"

type getUsersByIDsRes struct {
	Users []*mainflux.User
}
