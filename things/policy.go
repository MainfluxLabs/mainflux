package things

import (
	"context"
	"slices"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	ActionCommand = "command"
	ActionMessage = "message"
)

// typePolicy defines what a Thing type is allowed to do within its group.
type typePolicy struct {
	canCommandTo []string
	canMessageTo []string
}

// thingPolicies is the capability matrix per Thing type (same-group only).
// thingPolicies[publisherType] → allowed recipient types per action.
var thingPolicies = map[string]typePolicy{
	ThingTypeController: {
		canCommandTo: []string{ThingTypeSensor, ThingTypeActuator, ThingTypeDevice},
		canMessageTo: []string{ThingTypeSensor, ThingTypeActuator, ThingTypeDevice, ThingTypeController, ThingTypeGateway},
	},
	ThingTypeGateway: {
		canCommandTo: []string{ThingTypeSensor, ThingTypeActuator, ThingTypeDevice},
		canMessageTo: []string{ThingTypeSensor, ThingTypeActuator, ThingTypeDevice, ThingTypeController, ThingTypeGateway},
	},
	ThingTypeSensor: {
		canCommandTo: []string{},
		canMessageTo: []string{ThingTypeController, ThingTypeGateway, ThingTypeSensor},
	},
	ThingTypeActuator: {
		canCommandTo: []string{},
		canMessageTo: []string{ThingTypeController, ThingTypeGateway, ThingTypeActuator},
	},
	ThingTypeDevice: {
		canCommandTo: []string{ThingTypeDevice},
		canMessageTo: []string{ThingTypeDevice, ThingTypeController, ThingTypeGateway},
	},
}

func canCommand(_ context.Context, publisherType, recipientType string) error {
	policy, ok := thingPolicies[publisherType]
	if !ok || !slices.Contains(policy.canCommandTo, recipientType) {
		return errors.ErrAuthorization
	}
	return nil
}

func canMessage(_ context.Context, publisherType, recipientType string) error {
	policy, ok := thingPolicies[publisherType]
	if !ok || !slices.Contains(policy.canMessageTo, recipientType) {
		return errors.ErrAuthorization
	}
	return nil
}
