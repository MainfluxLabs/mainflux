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

// CanCommand checks the sending thing's type against a specific recipient type.
func CanCommand(thingType, recipientType string) error {
	allowed, ok := thingPolicies[thingType]
	if !ok || !slices.Contains(allowed, recipientType) {
		return errors.ErrAuthorization
	}
	return nil
}

// CanGroupCommand checks if the sending thing has any command authority.
// Group commands are intentionally coarse — filtering by recipient type
// is the responsibility of the receiving things, not the sender.
func CanGroupCommand(thingType string) error {
	if allowed := thingPolicies[thingType]; len(allowed) == 0 {
		return errors.ErrAuthorization
	}
	return nil
}
