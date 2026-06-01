// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/pkg/domain"

type listJSONMessagesRes struct {
	page domain.JSONMessagesPage
}

type listSenMLMessagesRes struct {
	page domain.SenMLMessagesPage
}
