// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package uiconfigs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

type Backup struct {
	OrgsConfigs   []OrgConfig
	ThingsConfigs []ThingConfig
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

	// Backup retrieves all org and thing configs.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds all orgs and things configs from a backup.
	Restore(ctx context.Context, token string, backup Backup) error
}

type configService struct {
	orgConfigs   OrgConfigRepository
	thingConfigs ThingConfigRepository
	things       protomfx.ThingsServiceClient
	auth         protomfx.AuthServiceClient
	idProvider   uuid.IDProvider
	logger       logger.Logger
}

var _ Service = (*configService)(nil)

func New(orgConfigs OrgConfigRepository, thingConfigs ThingConfigRepository, things protomfx.ThingsServiceClient, auth protomfx.AuthServiceClient, idp uuid.IDProvider, logger logger.Logger) Service {
	return &configService{
		orgConfigs:   orgConfigs,
		thingConfigs: thingConfigs,
		things:       things,
		auth:         auth,
		idProvider:   idp,
		logger:       logger,
	}
}

func (svc *configService) ViewOrgConfig(ctx context.Context, token, orgID string) (OrgConfig, error) {
	_, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return OrgConfig{}, err
	}

	if err := svc.canAccessOrg(ctx, token, orgID, auth.OrgSub, auth.Viewer); err != nil {
		return OrgConfig{}, err
	}

	return svc.orgConfigs.RetrieveByOrg(ctx, orgID)
}

func (svc *configService) ListOrgsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (OrgConfigPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.orgConfigs.RetrieveAll(ctx, pm)
	}

	if _, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return OrgConfigPage{}, err
	}

	all, err := svc.orgConfigs.RetrieveAll(ctx, pm)
	if err != nil {
		return OrgConfigPage{}, err
	}

	orgsConfigs := make([]OrgConfig, 0, len(all.OrgsConfigs))
	for _, oc := range all.OrgsConfigs {
		if err := svc.canAccessOrg(ctx, token, oc.OrgID, auth.OrgSub, auth.Viewer); err == nil {
			orgsConfigs = append(orgsConfigs, oc)
		}
	}

	return OrgConfigPage{
		Total:       uint64(len(orgsConfigs)),
		OrgsConfigs: orgsConfigs,
	}, nil
}

func (svc *configService) UpdateOrgConfig(ctx context.Context, token string, orgConfig OrgConfig) (OrgConfig, error) {
	_, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return OrgConfig{}, err
	}

	if err := svc.canAccessOrg(ctx, token, orgConfig.OrgID, auth.OrgSub, auth.Viewer); err != nil {
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

	if _, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return OrgConfigBackup{}, err
	}

	all, err := svc.orgConfigs.BackupAll(ctx)
	if err != nil {
		return OrgConfigBackup{}, err
	}

	orgsConfigs := make([]OrgConfig, 0, len(all.OrgsConfigs))
	for _, t := range all.OrgsConfigs {
		if err := svc.canAccessOrg(ctx, token, t.OrgID, auth.OrgSub, auth.Viewer); err == nil {
			orgsConfigs = append(orgsConfigs, t)
		}
	}
	return OrgConfigBackup{
		OrgsConfigs: orgsConfigs,
	}, nil
}

func (svc *configService) ViewThingConfig(ctx context.Context, token, thingID string) (ThingConfig, error) {
	_, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ThingConfig{}, err
	}

	if _, err := svc.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer}); err != nil {
		return ThingConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	return svc.thingConfigs.RetrieveByThing(ctx, thingID)
}

func (svc *configService) ListThingsConfigs(ctx context.Context, token string, pm apiutil.PageMetadata) (ThingConfigPage, error) {
	if err := svc.isAdmin(ctx, token); err == nil {
		return svc.thingConfigs.RetrieveAll(ctx, pm)
	}

	if _, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return ThingConfigPage{}, err
	}

	all, err := svc.thingConfigs.RetrieveAll(ctx, pm)
	if err != nil {
		return ThingConfigPage{}, err
	}

	thingsConfigs := make([]ThingConfig, 0, len(all.ThingsConfigs))
	for _, t := range all.ThingsConfigs {
		if _, err := svc.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: t.ThingID, Action: things.Viewer}); err == nil {
			thingsConfigs = append(thingsConfigs, t)
		}
	}

	return ThingConfigPage{
		Total:         uint64(len(thingsConfigs)),
		ThingsConfigs: thingsConfigs,
	}, nil
}

func (svc *configService) UpdateThingConfig(ctx context.Context, token string, thingConfig ThingConfig) (ThingConfig, error) {
	_, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return ThingConfig{}, err
	}

	if _, err := svc.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingConfig.ThingID, Action: things.Viewer}); err != nil {
		return ThingConfig{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	grID, err := svc.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: thingConfig.ThingID})
	if err != nil {
		return ThingConfig{}, err
	}

	thingConfig.GroupID = grID.GetValue()

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

	if _, err := svc.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return ThingConfigBackup{}, err
	}

	all, err := svc.thingConfigs.BackupAll(ctx)
	if err != nil {
		return ThingConfigBackup{}, err
	}

	thingsConfigs := make([]ThingConfig, 0, len(all.ThingsConfigs))
	for _, t := range all.ThingsConfigs {
		if _, err := svc.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: t.ThingID, Action: things.Viewer}); err == nil {
			thingsConfigs = append(thingsConfigs, t)
		}
	}

	return ThingConfigBackup{
		ThingsConfigs: thingsConfigs,
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

	return Backup{
		OrgsConfigs:   orgs.OrgsConfigs,
		ThingsConfigs: things.ThingsConfigs,
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

	return nil
}

func (svc *configService) canAccessOrg(ctx context.Context, token, orgID, subject, action string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Object:  orgID,
		Subject: subject,
		Action:  action,
	}

	if _, err := svc.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (svc *configService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: auth.RootSub,
	}

	if _, err := svc.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
