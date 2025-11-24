// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-redis/redis/v8"
)

const (
	streamID  = "mainflux.auth"
	streamLen = 1000
)

var _ auth.Service = (*eventStore)(nil)

type eventStore struct {
	svc    auth.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(svc auth.Service, client *redis.Client) auth.Service {
	return eventStore{
		svc:    svc,
		client: client,
	}
}

func (es eventStore) Identify(ctx context.Context, token string) (auth.Identity, error) {
	return es.svc.Identify(ctx, token)
}

func (es eventStore) Authorize(ctx context.Context, ar auth.AuthzReq) error {
	return es.svc.Authorize(ctx, ar)
}

func (es eventStore) AssignRole(ctx context.Context, id, role string) error {
	return es.svc.AssignRole(ctx, id, role)
}

func (es eventStore) RetrieveRole(ctx context.Context, id string) (string, error) {
	return es.svc.RetrieveRole(ctx, id)
}

func (es eventStore) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	sorg, err := es.svc.CreateOrg(ctx, token, org)
	if err != nil {
		return sorg, err
	}

	event := createOrgEvent{
		id: sorg.ID,
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return sorg, nil
}

func (es eventStore) UpdateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	return es.svc.UpdateOrg(ctx, token, org)
}

func (es eventStore) ViewOrg(ctx context.Context, token, id string) (auth.Org, error) {
	return es.svc.ViewOrg(ctx, token, id)
}

func (es eventStore) ListOrgs(ctx context.Context, token string, pm apiutil.PageMetadata) (auth.OrgsPage, error) {
	return es.svc.ListOrgs(ctx, token, pm)
}

func (es eventStore) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.svc.RemoveOrgs(ctx, token, id); err != nil {
			return err
		}

		event := removeOrgEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       streamID,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}

func (es eventStore) GetOwnerIDByOrgID(ctx context.Context, orgID string) (string, error) {
	return es.svc.GetOwnerIDByOrgID(ctx, orgID)
}

func (es eventStore) Backup(ctx context.Context, token string) (auth.Backup, error) {
	return es.svc.Backup(ctx, token)
}

func (es eventStore) Restore(ctx context.Context, token string, backup auth.Backup) error {
	return es.svc.Restore(ctx, token, backup)
}

func (es eventStore) CreateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) error {
	return es.svc.CreateOrgMemberships(ctx, token, orgID, oms...)
}

func (es eventStore) RemoveOrgMemberships(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	return es.svc.RemoveOrgMemberships(ctx, token, orgID, memberIDs...)
}

func (es eventStore) UpdateOrgMemberships(ctx context.Context, token, orgID string, oms ...auth.OrgMembership) error {
	return es.svc.UpdateOrgMemberships(ctx, token, orgID, oms...)
}

func (es eventStore) ListOrgMemberships(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (auth.OrgMembershipsPage, error) {
	return es.svc.ListOrgMemberships(ctx, token, orgID, pm)
}

func (es eventStore) ViewOrgMembership(ctx context.Context, token, orgID, memberID string) (auth.OrgMembership, error) {
	return es.svc.ViewOrgMembership(ctx, token, orgID, memberID)
}

func (es eventStore) BackupOrgMemberships(ctx context.Context, token string, orgID string) (auth.OrgMembershipsBackup, error) {
	return es.svc.BackupOrgMemberships(ctx, token, orgID)
}

func (es eventStore) RestoreOrgMemberships(ctx context.Context, token string, orgID string, backup auth.OrgMembershipsBackup) error {
	return es.svc.RestoreOrgMemberships(ctx, token, orgID, backup)
}

func (es eventStore) CreateOrgInvite(ctx context.Context, token, email, role, orgID, invRedirectPath string) (auth.OrgInvite, error) {
	return es.svc.CreateOrgInvite(ctx, token, email, role, orgID, invRedirectPath)
}

func (es eventStore) CreateDormantOrgInvite(ctx context.Context, token, orgID, role, platformInviteID string) (auth.OrgInvite, error) {
	return es.svc.CreateDormantOrgInvite(ctx, token, orgID, role, platformInviteID)
}

func (es eventStore) RevokeOrgInvite(ctx context.Context, token, inviteID string) error {
	return es.svc.RevokeOrgInvite(ctx, token, inviteID)
}

func (es eventStore) RespondOrgInvite(ctx context.Context, token, inviteID string, accept bool) error {
	return es.svc.RespondOrgInvite(ctx, token, inviteID, accept)
}

func (es eventStore) ActivateOrgInvite(ctx context.Context, platformInviteID, userID, invRedirectPath string) error {
	return es.svc.ActivateOrgInvite(ctx, platformInviteID, userID, invRedirectPath)
}

func (es eventStore) ViewOrgInvite(ctx context.Context, token, inviteID string) (auth.OrgInvite, error) {
	return es.svc.ViewOrgInvite(ctx, token, inviteID)
}

func (es eventStore) ListOrgInvitesByUser(ctx context.Context, token, userType, userID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	return es.svc.ListOrgInvitesByUser(ctx, token, userType, userID, pm)
}

func (es eventStore) ListOrgInvitesByOrg(ctx context.Context, token, orgID string, pm auth.PageMetadataInvites) (auth.OrgInvitesPage, error) {
	return es.svc.ListOrgInvitesByOrg(ctx, token, orgID, pm)
}

func (es eventStore) SendOrgInviteEmail(ctx context.Context, invite auth.OrgInvite, email, orgName, invRedirectPath string) error {
	return es.svc.SendOrgInviteEmail(ctx, invite, email, orgName, invRedirectPath)
}

func (es eventStore) Issue(ctx context.Context, token string, key auth.Key) (auth.Key, string, error) {
	return es.svc.Issue(ctx, token, key)
}

func (es eventStore) Revoke(ctx context.Context, token, id string) error {
	return es.svc.Revoke(ctx, token, id)
}

func (es eventStore) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	return es.svc.RetrieveKey(ctx, token, id)
}
