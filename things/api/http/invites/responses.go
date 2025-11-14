package invites

import (
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
)

type createGroupInviteRes struct {
	ID      string
	created bool
}

func (res createGroupInviteRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/invites/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createGroupInviteRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createGroupInviteRes) Empty() bool {
	return false
}

type revokeGroupInviteRes struct{}

func (res revokeGroupInviteRes) Headers() map[string]string {
	return nil
}

func (res revokeGroupInviteRes) Code() int {
	return http.StatusNoContent
}

func (res revokeGroupInviteRes) Empty() bool {
	return true
}

type respondGroupInviteRes struct {
	accept bool
}

func (res respondGroupInviteRes) Headers() map[string]string {
	return nil
}

func (res respondGroupInviteRes) Code() int {
	if res.accept {
		return http.StatusCreated
	}

	return http.StatusNoContent
}

func (res respondGroupInviteRes) Empty() bool {
	return true
}

type groupInviteRes struct {
	invites.InviteRes
	GroupID   string `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
}

type groupInvitePageRes struct {
	invites.PageRes
	Invites []groupInviteRes `json:"invites"`
}

func buildGroupInvitesPageRes(page things.GroupInvitesPage, pm invites.PageMetadataInvites) groupInvitePageRes {
	res := groupInvitePageRes{
		PageRes: invites.PageRes{
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Total:  page.Total,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			State:  pm.State,
		},
		Invites: make([]groupInviteRes, 0, len(page.Invites)),
	}

	for _, inv := range page.Invites {
		res.Invites = append(res.Invites, buildGroupInviteRes(inv))
	}

	return res
}

func buildGroupInviteRes(inv things.GroupInvite) groupInviteRes {
	var inviteeID string

	// Only return "invitee_id" property in response if InviteeID is non-NULL
	if inv.InviteeID.Valid {
		inviteeID = inv.InviteeID.String
	}

	return groupInviteRes{
		InviteRes: invites.InviteRes{
			ID:           inv.ID,
			InviteeID:    inviteeID,
			InviteeEmail: inv.InviteeEmail,
			InviteeRole:  inv.InviteeRole,
			InviterID:    inv.InviterID,
			InviterEmail: inv.InviterEmail,
			CreatedAt:    inv.CreatedAt,
			ExpiresAt:    inv.ExpiresAt,
			State:        inv.State,
		},
		GroupID:   inv.GroupID,
		GroupName: inv.GroupName,
	}
}
