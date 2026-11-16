// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	auth.Service
	pub events.Publisher
}

func NewEventStoreMiddleware(svc auth.Service, pub events.Publisher) auth.Service {
	return eventStore{
		Service: svc,
		pub:     pub,
	}
}

func (es eventStore) Issue(ctx context.Context, token string, key auth.Key) (auth.Key, string, error) {
	issuedKey, secret, err := es.Service.Issue(ctx, token, key)
	if err != nil {
		return issuedKey, secret, err
	}

	if key.Type == auth.APIKey {
		es.pub.Publish(ctx, events.Event{
			Action: events.APIKeyIssued{
				ID:        issuedKey.ID,
				ExpiresAt: issuedKey.ExpiresAt,
			},
		})
	}

	if key.Impersonator != nil {
		es.pub.Publish(ctx, events.Event{
			Action: events.UserImpersonated{
				ID:    key.IssuerID,
				Email: key.Subject,
			},
			JWTUserIdentity: *key.Impersonator,
		})
	}

	return issuedKey, secret, nil
}

func (es eventStore) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	sorg, err := es.Service.CreateOrg(ctx, token, org)
	if err != nil {
		return sorg, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.OrgCreated{ID: sorg.ID},
		OrgID:  sorg.ID,
	})

	return sorg, nil
}

func (es eventStore) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveOrgs(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.Event{
			Action: events.OrgRemoved{ID: id},
			OrgID:  id,
		})
	}

	return nil
}

func (es eventStore) UpdateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	updatedOrg, err := es.Service.UpdateOrg(ctx, token, org)
	if err != nil {
		return updatedOrg, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.OrgUpdated{
			ID:          updatedOrg.ID,
			Name:        updatedOrg.Name,
			Description: updatedOrg.Description,
			Metadata:    updatedOrg.Metadata,
		},
		OrgID: updatedOrg.ID,
	})

	return updatedOrg, nil
}

func (es eventStore) CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) ([]auth.OrgMembership, error) {
	saved, err := es.Service.CreateOrgMemberships(ctx, token, orgID, oms...)
	if err != nil {
		return saved, err
	}

	for _, om := range saved {
		es.pub.Publish(ctx, events.Event{
			Action: events.OrgMembershipCreated{
				OrgID:    om.OrgID,
				MemberID: om.MemberID,
				Role:     om.Role,
			},
			OrgID: om.OrgID,
		})
	}

	return saved, nil
}

func (es eventStore) UpdateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) ([]auth.OrgMembership, error) {
	updated, err := es.Service.UpdateOrgMemberships(ctx, token, orgID, oms...)
	if err != nil {
		return updated, err
	}

	for _, om := range updated {
		es.pub.Publish(ctx, events.Event{
			Action: events.OrgMembershipUpdated{
				OrgID:    om.OrgID,
				MemberID: om.MemberID,
				Role:     om.Role,
			},
			OrgID: om.OrgID,
		})
	}

	return updated, nil
}

func (es eventStore) RemoveOrgMemberships(ctx context.Context, token, orgID string, memberIDs ...string) error {
	if err := es.Service.RemoveOrgMemberships(ctx, token, orgID, memberIDs...); err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		es.pub.Publish(ctx, events.Event{
			Action: events.OrgMembershipRemoved{
				OrgID:    orgID,
				MemberID: memberID,
			},
			OrgID: orgID,
		})
	}

	return nil
}

func (es eventStore) CreateOrgInvite(ctx context.Context, token string, oi auth.OrgInvite, invRedirectPath string) (auth.OrgInvite, error) {
	invite, err := es.Service.CreateOrgInvite(ctx, token, oi, invRedirectPath)
	if err != nil {
		return invite, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.OrgInviteCreated{
			ID:           invite.ID,
			InviteeID:    invite.InviteeID,
			InviteeEmail: invite.InviteeEmail,
			InviteeRole:  invite.InviteeRole,
			GroupInvites: invite.GroupInvites,
		},
		OrgID: invite.OrgID,
	})

	return invite, nil
}

func (es eventStore) RevokeOrgInvite(ctx context.Context, token, inviteID string) (auth.OrgInvite, error) {
	invite, err := es.Service.RevokeOrgInvite(ctx, token, inviteID)
	if err != nil {
		return invite, err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.OrgInviteRevoked{ID: invite.ID},
		OrgID:  invite.OrgID,
	})

	return invite, nil
}

func (es eventStore) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) (auth.OrgInvite, error) {
	invite, err := es.Service.RespondOrgInvite(ctx, token, inviteID, accept)
	if err != nil {
		return invite, err
	}

	var action events.EventAction = events.OrgInviteDeclined{ID: invite.ID}
	if accept {
		action = events.OrgInviteAccepted{ID: invite.ID}
	}

	es.pub.Publish(ctx, events.Event{
		Action: action,
		OrgID:  invite.OrgID,
	})

	return invite, nil
}

func (es eventStore) Revoke(ctx context.Context, token, id string) error {
	if err := es.Service.Revoke(ctx, token, id); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.APIKeyRevoked{ID: id},
	})

	return nil
}

func (es eventStore) LogoutAll(ctx context.Context, token string) error {
	if err := es.Service.LogoutAll(ctx, token); err != nil {
		return err
	}

	es.pub.Publish(ctx, events.Event{
		Action: events.SessionLoggedOutAll{},
	})

	return nil
}
