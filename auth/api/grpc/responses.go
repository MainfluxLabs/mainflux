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

type ownerIDByOrgIDRes struct {
	ownerID string
}
