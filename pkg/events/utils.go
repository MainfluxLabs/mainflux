package events

const (
	thingPrefix   = "thing."
	profilePrefix = "profile."
	groupPrefix   = "group."

	ThingCreate                = thingPrefix + "create"
	ThingUpdate                = thingPrefix + "update"
	ThingUpdateGroupAndProfile = thingPrefix + "update_group_and_profile"
	ThingRemove                = thingPrefix + "remove"

	ProfileCreate = profilePrefix + "create"
	ProfileUpdate = profilePrefix + "update"
	ProfileRemove = profilePrefix + "remove"

	GroupRemove = groupPrefix + "remove"
)

func Read(event map[string]any, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
