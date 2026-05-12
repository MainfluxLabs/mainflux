package events

type removeThingEvent struct {
	id string
}

type removeGroupEvent struct {
	thingIDs []string
}
