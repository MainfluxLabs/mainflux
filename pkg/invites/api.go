package invites

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// ErrInvalidInviteResponse indicates an invalid Invite response action string.
var ErrInvalidInviteResponse = errors.New("invalid invite response action")

const (
	ResponseActionKey = "action"
	StateKey          = "state"
)

func BuildPageMetadataInvites(r *http.Request) (PageMetadataInvites, error) {
	pm := PageMetadataInvites{}

	apm, err := apiutil.BuildPageMetadata(r)
	if err != nil {
		return PageMetadataInvites{}, err
	}

	pm.PageMetadata = apm

	state, err := apiutil.ReadStringQuery(r, StateKey, "")
	if err != nil {
		return PageMetadataInvites{}, err
	}

	pm.State = state

	return pm, nil
}

type PageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Ord    string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	State  string `json:"state,omitempty"`
}

type InviteRes struct {
	ID           string    `json:"id"`
	InviteeID    string    `json:"invitee_id"`
	InviteeEmail string    `json:"invitee_email,omitempty"`
	InviteeRole  string    `json:"invitee_role"`
	InviterID    string    `json:"inviter_id,omitempty"`
	InviterEmail string    `json:"inviter_email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	State        string    `json:"state"`
}
