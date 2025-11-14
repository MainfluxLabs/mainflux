package postgres

import (
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
)

type groupInviteRepository struct {
	*invites.CommonInviteRepository[things.GroupInvite]
}

func NewGroupInviteRepository(db dbutil.Database) things.GroupInviteRepository {
	return &groupInviteRepository{
		CommonInviteRepository: invites.NewCommonInviteRepository[things.GroupInvite](db),
	}
}
