package invites

import (
	"database/sql"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	UserTypeInvitee = "invitee"
	UserTypeInviter = "inviter"

	InviteStatePending  = "pending"
	InviteStateExpired  = "expired"
	InviteStateRevoked  = "revoked"
	InviteStateAccepted = "accepted"
	InviteStateDeclined = "declined"
)

type Invitable interface {
	GetCommon() InviteCommon

	// GetDestinationID returns the unique ID of the destination that the invite belongs to.
	// For Organization invites, it's the Organization ID, and for Group invites, it's the Group ID.
	GetDestinationID() string

	// Get the name of the database column storing the destination ID.
	ColumnDestinationID() string

	// Get the name of the DB table storing the invite of this type
	TableName() string

	ToDBInvite() DbInvite
}

type InviteCommon struct {
	ID           string         `db:"id"`
	InviteeID    sql.NullString `db:"invitee_id"`
	InviteeEmail string
	InviterID    string `db:"inviter_id"`
	InviterEmail string
	// The invitee's role in the destination: for Orgs this is the role in the Org,
	// and for Groups it's the role in the Group.
	InviteeRole string    `db:"invitee_role"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
	State       string    `db:"state"`
}

func (invite InviteCommon) ToDBInvite() DbInvite {
	return DbInvite{
		ID:          invite.ID,
		InviteeID:   invite.InviteeID,
		InviterID:   invite.InviterID,
		InviteeRole: invite.InviteeRole,
		CreatedAt:   invite.CreatedAt,
		ExpiresAt:   invite.ExpiresAt,
		State:       invite.State,
	}
}

type PageMetadataInvites struct {
	apiutil.PageMetadata
	State string `json:"state,omitempty"`
}

type InvitesPage[T Invitable] struct {
	Invites []T
	Total   uint64
}
