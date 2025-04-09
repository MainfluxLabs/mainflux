// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies an API for managing rules.
type Service interface {
	// CreateRules creates rules.
	CreateRules(ctx context.Context, token string, rules ...Rule) ([]Rule, error)

	// ListRulesByProfile retrieves a paginated list of rules by profile.
	ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ListRulesByGroup retrieves a paginated list of rules by group.
	ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ViewRule retrieves a specific rule by its ID.
	ViewRule(ctx context.Context, token, id string) (Rule, error)

	// UpdateRule updates the rule identified by the provided ID.
	UpdateRule(ctx context.Context, token string, rule Rule) error

	// RemoveRules removes the rules identified with the provided IDs.
	RemoveRules(ctx context.Context, token string, ids ...string) error

	consumers.Consumer
}

type rulesService struct {
	rules      RuleRepository
	things     protomfx.ThingsServiceClient
	idProvider uuid.IDProvider
}

var _ Service = (*rulesService)(nil)

// New instantiates the rules service implementation.
func New(rules RuleRepository, things protomfx.ThingsServiceClient, idp uuid.IDProvider) Service {
	return &rulesService{
		rules:      rules,
		things:     things,
		idProvider: idp,
	}
}

func (rs *rulesService) CreateRules(ctx context.Context, token string, rules ...Rule) ([]Rule, error) {
	var rls []Rule
	for _, rule := range rules {
		r, err := rs.createRule(ctx, &rule, token)
		if err != nil {
			return []Rule{}, err
		}
		rls = append(rls, r)
	}

	return rls, nil
}

func (rs *rulesService) createRule(ctx context.Context, rule *Rule, token string) (Rule, error) {
	_, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.ProfileID, Action: things.Editor})
	if err != nil {
		return Rule{}, err
	}

	//TODO: Replace by GetGroupIDByProfileID
	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: rule.ProfileID})
	if err != nil {
		return Rule{}, err
	}
	rule.GroupID = grID.GetValue()

	id, err := rs.idProvider.ID()
	if err != nil {
		return Rule{}, err
	}
	rule.ID = id

	rls, err := rs.rules.Save(ctx, *rule)
	if err != nil {
		return Rule{}, err
	}

	if len(rls) == 0 {
		return Rule{}, errors.ErrCreateEntity
	}

	return rls[0], nil
}

func (rs *rulesService) ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (RulesPage, error) {
	_, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: profileID, Action: things.Viewer})
	if err != nil {
		return RulesPage{}, err
	}

	rules, err := rs.rules.RetrieveByProfile(ctx, profileID, pm)
	if err != nil {
		return RulesPage{}, err
	}

	return rules, nil
}

func (rs *rulesService) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (RulesPage, error) {
	_, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return RulesPage{}, err
	}

	rules, err := rs.rules.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return RulesPage{}, err
	}

	return rules, nil
}

func (rs *rulesService) ViewRule(ctx context.Context, token, id string) (Rule, error) {
	rule, err := rs.rules.RetrieveByID(ctx, id)
	if err != nil {
		return Rule{}, err
	}
	//TODO Replace by CanUserAccessGroup
	if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.ProfileID, Action: things.Viewer}); err != nil {
		return Rule{}, err
	}

	return rule, nil
}

func (rs *rulesService) UpdateRule(ctx context.Context, token string, rule Rule) error {
	r, err := rs.rules.RetrieveByID(ctx, rule.ID)
	if err != nil {
		return err
	}
	//TODO Replace by CanUserAccessGroup
	if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: r.ProfileID, Action: things.Editor}); err != nil {
		return err
	}

	return rs.rules.Update(ctx, rule)
}

func (rs *rulesService) RemoveRules(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		webhook, err := rs.rules.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		//TODO Replace by CanUserAccessGroup
		if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: webhook.ProfileID, Action: things.Editor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}
	return rs.rules.Remove(ctx, ids...)
}

func (rs *rulesService) Consume(messages interface{}) error {
	//TODO Implement Consume
	panic("implement me")
}
