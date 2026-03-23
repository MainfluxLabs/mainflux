package things

import (
	"slices"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// thingPolicies is the command capability matrix per Thing type (same-group only).
// Any type can send messages to any other type within the same group.
var thingPolicies = map[string][]string{
	ThingTypeController: {ThingTypeSensor, ThingTypeActuator, ThingTypeDevice},
	ThingTypeGateway:    {ThingTypeSensor, ThingTypeActuator, ThingTypeDevice},
	ThingTypeDevice:     {ThingTypeDevice},
	ThingTypeSensor:     {},
	ThingTypeActuator:   {},
}

// CanCommand checks publisher type against a specific recipient type.
func CanCommand(publisherType, recipientType string) error {
	allowed, ok := thingPolicies[publisherType]
	if !ok || !slices.Contains(allowed, recipientType) {
		return errors.ErrAuthorization
	}
	return nil
}

// CanGroupCommand checks if the publisher has any command authority,
// with no recipient type since the command broadcasts to all things in a group.
func CanGroupCommand(publisherType string) error {
	if allowed := thingPolicies[publisherType]; len(allowed) == 0 {
		return errors.ErrAuthorization
	}
	return nil
}
