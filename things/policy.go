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

func CanCommand(publisherType, recipientType string) error {
	allowed, ok := thingPolicies[publisherType]
	if !ok || !slices.Contains(allowed, recipientType) {
		return errors.ErrAuthorization
	}
	return nil
}

// CanGroupCommand reports whether a thing of publisherType may issue commands
// to an entire group.
func CanGroupCommand(publisherType string) error {
	if allowed := thingPolicies[publisherType]; len(allowed) == 0 {
		return errors.ErrAuthorization
	}
	return nil
}
