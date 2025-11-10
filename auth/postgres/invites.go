// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
)

type orgInviteRepository struct {
	*invites.CommonInviteRepository[auth.OrgInvite]
}

func NewOrgInviteRepository(db dbutil.Database) auth.OrgInviteRepository {
	return &orgInviteRepository{
		CommonInviteRepository: invites.NewCommonInviteRepository[auth.OrgInvite](db),
	}
}
