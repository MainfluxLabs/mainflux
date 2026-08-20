// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

type Backup struct {
	OrgsConfigs   []OrgConfig
	ThingsConfigs []ThingConfig
	GroupsConfigs []GroupConfig
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ViewOrgConfig retrieves the org config for the authenticated user and org.
	ViewOrgConfig(ctx context.Context, token, orgID string) (OrgConfig, error)

	// ListOrgsConfigs retrieves all org configs.
	ListOrgsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (OrgConfigPage, error)

	// UpdateOrgConfig updates an existing org config for the authenticated user.
	UpdateOrgConfig(ctx context.Context, token string, orgConfig OrgConfig) (OrgConfig, error)

	// RemoveOrgConfig removes the org config by org id.
	RemoveOrgConfig(ctx context.Context, orgID string) error

	// BackupOrgsConfigs retrieves all org configs.
	BackupOrgsConfigs(ctx context.Context, token string) (OrgConfigBackup, error)

	// ViewThingConfig retrieves the thing config for the authenticated user and thing.
	ViewThingConfig(ctx context.Context, token, thingID string) (ThingConfig, error)

	// ListThingsConfigs retrieves all thing configs.
	ListThingsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (ThingConfigPage, error)

	// UpdateThingConfig updates an existing thing config for the authenticated user.
	UpdateThingConfig(ctx context.Context, token string, thingConfig ThingConfig) (ThingConfig, error)

	// RemoveThingConfig removes the thing config by thing id.
	RemoveThingConfig(ctx context.Context, thingID string) error

	// RemoveThingConfigByGroup removes the thing config by group id.
	RemoveThingConfigByGroup(ctx context.Context, groupID string) error

	// BackupThingsConfigs retrieves all thing configs.
	BackupThingsConfigs(ctx context.Context, token string) (ThingConfigBackup, error)

	// ViewGroupConfig retrieves the group config for the authenticated user and group.
	ViewGroupConfig(ctx context.Context, token, groupID string) (GroupConfig, error)

	// ListGroupsConfigs retrieves all group configs.
	ListGroupsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (GroupConfigPage, error)

	// UpdateGroupConfig updates an existing group config for the authenticated user.
	UpdateGroupConfig(ctx context.Context, token string, groupConfig GroupConfig) (GroupConfig, error)

	// RemoveGroupConfig removes the group config by group id.
	RemoveGroupConfig(ctx context.Context, groupID string) error

	// BackupGroupsConfigs retrieves all group configs.
	BackupGroupsConfigs(ctx context.Context, token string) (GroupConfigBackup, error)

	// Backup retrieves all org, thing and group configs.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds all orgs, things and groups configs from a backup.
	Restore(ctx context.Context, token string, backup Backup) error
}

type configService struct {
	orgConfigs   OrgConfigRepository
	thingConfigs ThingConfigRepository
	groupConfigs GroupConfigRepository
	things       domain.ThingsClient
	auth         domain.AuthClient
	idProvider   uuid.IDProvider
	logger       logger.Logger
}

var _ Service = (*configService)(nil)

func New(orgConfigs OrgConfigRepository, thingConfigs ThingConfigRepository, groupConfigs GroupConfigRepository, things domain.ThingsClient, auth domain.AuthClient, idp uuid.IDProvider, logger logger.Logger) Service {
	return &configService{
		orgConfigs:   orgConfigs,
		thingConfigs: thingConfigs,
		groupConfigs: groupConfigs,
		things:       things,
		auth:         auth,
		idProvider:   idp,
		logger:       logger,
	}
}

func (svc *configService) ViewOrgConfig(ctx context.Context, token, orgID string) (OrgConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return OrgConfig{}, err
	}

	if err := svc.canAccessOrg(ctx, token, orgID, domain.OrgSub, domain.OrgViewer); err != nil {
		return OrgConfig{}, err
	}

	return svc.orgConfigs.RetrieveByOrg(ctx, orgID)
}

func (svc *configService) ListOrgsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (OrgConfigPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgConfigs.RetrieveAll(ctx, pm)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return OrgConfigPage{}, err
	}

	all, err := svc.orgConfigs.RetrieveAll(ctx, pm)
	if err != nil {
		return OrgConfigPage{}, err
	}

	orgsConfigs := make([]OrgConfig, 0, len(all.OrgsConfigs))
	for _, oc := range all.OrgsConfigs {
		if err := svc.canAccessOrg(ctx, token, oc.OrgID, domain.OrgSub, domain.OrgViewer); err == nil {
			orgsConfigs = append(orgsConfigs, oc)
		}
	}

	return OrgConfigPage{
		Total:       uint64(len(orgsConfigs)),
		OrgsConfigs: orgsConfigs,
	}, nil
}

func (svc *configService) UpdateOrgConfig(ctx context.Context, token string, orgConfig OrgConfig) (OrgConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return OrgConfig{}, err
	}

	if err := svc.canAccessOrg(ctx, token, orgConfig.OrgID, domain.OrgSub, domain.OrgEditor); err != nil {
		return OrgConfig{}, err
	}

	updated, err := svc.orgConfigs.Update(ctx, orgConfig)
	if err != nil {
		return OrgConfig{}, err
	}

	return updated, nil
}

func (svc *configService) RemoveOrgConfig(ctx context.Context, orgID string) error {
	return svc.orgConfigs.Remove(ctx, orgID)
}

func (svc *configService) BackupOrgsConfigs(ctx context.Context, token string) (OrgConfigBackup, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgConfigs.BackupAll(ctx)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return OrgConfigBackup{}, err
	}

	all, err := svc.orgConfigs.BackupAll(ctx)
	if err != nil {
		return OrgConfigBackup{}, err
	}

	orgsConfigs := make([]OrgConfig, 0, len(all.OrgsConfigs))
	for _, t := range all.OrgsConfigs {
		if err := svc.canAccessOrg(ctx, token, t.OrgID, domain.OrgSub, domain.OrgViewer); err == nil {
			orgsConfigs = append(orgsConfigs, t)
		}
	}
	return OrgConfigBackup{
		OrgsConfigs: orgsConfigs,
	}, nil
}

func (svc *configService) ViewThingConfig(ctx context.Context, token, thingID string) (ThingConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return ThingConfig{}, err
	}

	if err := svc.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer}); err != nil {
		return ThingConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.thingConfigs.RetrieveByThing(ctx, thingID)
}

func (svc *configService) ListThingsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (ThingConfigPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.thingConfigs.RetrieveAll(ctx, pm)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return ThingConfigPage{}, err
	}

	all, err := svc.thingConfigs.RetrieveAll(ctx, pm)
	if err != nil {
		return ThingConfigPage{}, err
	}

	thingsConfigs := make([]ThingConfig, 0, len(all.ThingsConfigs))
	for _, t := range all.ThingsConfigs {
		if err := svc.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: t.ThingID, Action: domain.GroupViewer}); err == nil {
			thingsConfigs = append(thingsConfigs, t)
		}
	}

	return ThingConfigPage{
		Total:         uint64(len(thingsConfigs)),
		ThingsConfigs: thingsConfigs,
	}, nil
}

func (svc *configService) UpdateThingConfig(ctx context.Context, token string, thingConfig ThingConfig) (ThingConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return ThingConfig{}, err
	}

	if err := svc.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingConfig.ThingID, Action: domain.GroupViewer}); err != nil {
		return ThingConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	groupID, err := svc.things.GetGroupIDByThing(ctx, thingConfig.ThingID)
	if err != nil {
		return ThingConfig{}, err
	}

	thingConfig.GroupID = groupID

	updated, err := svc.thingConfigs.Update(ctx, thingConfig)
	if err != nil {
		return ThingConfig{}, err
	}

	return updated, nil
}

func (svc *configService) RemoveThingConfig(ctx context.Context, thingID string) error {
	return svc.thingConfigs.Remove(ctx, thingID)
}

func (svc *configService) RemoveThingConfigByGroup(ctx context.Context, groupID string) error {
	return svc.thingConfigs.RemoveByGroup(ctx, groupID)
}

func (svc *configService) BackupThingsConfigs(ctx context.Context, token string) (ThingConfigBackup, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.thingConfigs.BackupAll(ctx)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return ThingConfigBackup{}, err
	}

	all, err := svc.thingConfigs.BackupAll(ctx)
	if err != nil {
		return ThingConfigBackup{}, err
	}

	thingsConfigs := make([]ThingConfig, 0, len(all.ThingsConfigs))
	for _, t := range all.ThingsConfigs {
		if err := svc.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: t.ThingID, Action: domain.GroupViewer}); err == nil {
			thingsConfigs = append(thingsConfigs, t)
		}
	}

	return ThingConfigBackup{
		ThingsConfigs: thingsConfigs,
	}, nil
}

func (svc *configService) ViewGroupConfig(ctx context.Context, token, groupID string) (GroupConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return GroupConfig{}, err
	}

	if err := svc.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return GroupConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.groupConfigs.RetrieveByGroup(ctx, groupID)
}

func (svc *configService) ListGroupsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (GroupConfigPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.groupConfigs.RetrieveAll(ctx, pm)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return GroupConfigPage{}, err
	}

	all, err := svc.groupConfigs.RetrieveAll(ctx, pm)
	if err != nil {
		return GroupConfigPage{}, err
	}

	groupsConfigs := make([]GroupConfig, 0, len(all.GroupsConfigs))
	for _, g := range all.GroupsConfigs {
		if err := svc.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: g.GroupID, Action: domain.GroupViewer}); err == nil {
			groupsConfigs = append(groupsConfigs, g)
		}
	}

	return GroupConfigPage{
		Total:         uint64(len(groupsConfigs)),
		GroupsConfigs: groupsConfigs,
	}, nil
}

func (svc *configService) UpdateGroupConfig(ctx context.Context, token string, groupConfig GroupConfig) (GroupConfig, error) {
	_, err := svc.auth.Identify(ctx, token)
	if err != nil {
		return GroupConfig{}, err
	}

	if err := svc.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupConfig.GroupID, Action: domain.GroupEditor}); err != nil {
		return GroupConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	updated, err := svc.groupConfigs.Update(ctx, groupConfig)
	if err != nil {
		return GroupConfig{}, err
	}

	return updated, nil
}

func (svc *configService) RemoveGroupConfig(ctx context.Context, groupID string) error {
	return svc.groupConfigs.Remove(ctx, groupID)
}

func (svc *configService) BackupGroupsConfigs(ctx context.Context, token string) (GroupConfigBackup, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.groupConfigs.BackupAll(ctx)
	}

	if _, err := svc.auth.Identify(ctx, token); err != nil {
		return GroupConfigBackup{}, err
	}

	all, err := svc.groupConfigs.BackupAll(ctx)
	if err != nil {
		return GroupConfigBackup{}, err
	}

	groupsConfigs := make([]GroupConfig, 0, len(all.GroupsConfigs))
	for _, g := range all.GroupsConfigs {
		if err := svc.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: g.GroupID, Action: domain.GroupViewer}); err == nil {
			groupsConfigs = append(groupsConfigs, g)
		}
	}

	return GroupConfigBackup{
		GroupsConfigs: groupsConfigs,
	}, nil
}

func (svc *configService) Backup(ctx context.Context, token string) (Backup, error) {
	orgs, err := svc.BackupOrgsConfigs(ctx, token)
	if err != nil {
		return Backup{}, err
	}

	things, err := svc.BackupThingsConfigs(ctx, token)
	if err != nil {
		return Backup{}, err
	}

	groups, err := svc.BackupGroupsConfigs(ctx, token)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		OrgsConfigs:   orgs.OrgsConfigs,
		ThingsConfigs: things.ThingsConfigs,
		GroupsConfigs: groups.GroupsConfigs,
	}, nil
}

func (svc *configService) Restore(ctx context.Context, token string, backup Backup) error {
	for _, orgConfig := range backup.OrgsConfigs {
		if _, err := svc.orgConfigs.Save(ctx, orgConfig); err != nil {
			return err
		}
	}

	for _, thingConfig := range backup.ThingsConfigs {
		if _, err := svc.thingConfigs.Save(ctx, thingConfig); err != nil {
			return err
		}
	}

	for _, groupConfig := range backup.GroupsConfigs {
		if _, err := svc.groupConfigs.Save(ctx, groupConfig); err != nil {
			return err
		}
	}

	return nil
}

func (svc *configService) canAccessOrg(ctx context.Context, token, orgID, subject, action string) error {
	if err := svc.auth.Authorize(ctx, domain.AuthzReq{Token: token, Object: orgID, Subject: subject, Action: action}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (svc *configService) isAdmin(ctx context.Context, token string) error {
	if err := svc.auth.Authorize(ctx, domain.AuthzReq{Token: token, Subject: domain.RootSub}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
