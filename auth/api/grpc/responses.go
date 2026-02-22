// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/auth"

type identityRes struct {
	id    string
	email string
}

type issueRes struct {
	value string
}

type emptyRes struct {
	err error
}

type retrieveRoleRes struct {
	role string
}

type ownerIDByOrgRes struct {
	ownerID string
}

type orgRes struct {
	id      string
	ownerID string
	name    string
}

type orgInviteRes struct {
	auth.OrgInvite
}
