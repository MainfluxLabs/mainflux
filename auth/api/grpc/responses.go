// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

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

type ownerIDRes struct {
	ownerID string
}

type orgMembershipRes struct {
	orgID    string
	memberID string
	role     string
}

type orgRes struct {
	id          string
	ownerID     string
	name        string
	description string
}
