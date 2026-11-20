// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/ota"
	otaevents "github.com/MainfluxLabs/mainflux/ota/events"
	otamocks "github.com/MainfluxLabs/mainflux/ota/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "admin@example.com"
	thingID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID      = "574106f7-030e-4881-8ab0-151195c29f94"
	otherGroupID = "674106f7-030e-4881-8ab0-151195c29f95"
	profileID    = "774106f7-030e-4881-8ab0-151195c29f96"
)

var firmware = ota.Update{
	FileName: "fw-sensor-1.4.0.bin",
	Version:  "1.4.0",
	FileSize: 412,
	Checksum: "9f2c4a0e7b1d3c5f6a8e9b0c2d4f6a8b0c2e4f6a8b0c2d4e6f8a0b2c4d6ea71e",
	Timeout:  time.Hour,
}

func TestHandleThingGroupAndProfileUpdated(t *testing.T) {
	cases := []struct {
		desc       string
		newGroupID string
		size       int
	}{
		{
			desc:       "handle profile change in the same group",
			newGroupID: groupID,
			size:       1,
		},
		{
			desc:       "handle move to another group",
			newGroupID: otherGroupID,
			size:       0,
		},
	}

	for _, tc := range cases {
		things := map[string]domain.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		}
		groups := map[string]domain.Group{token: {ID: groupID}}
		tcl := mocks.NewThingsServiceClient(nil, things, groups)
		svc := ota.New(tcl, otamocks.NewUpdateRepository(), otamocks.NewCommandPublisher(), uuid.NewMock(), "https://localhost/fs/groupfiles", logger.NewMock())
		handler := otaevents.NewEventHandler(svc)

		u, err := svc.CreateUpdate(context.Background(), token, groupID, firmware, thingID)
		require.Nil(t, err, fmt.Sprintf("%s: creating update failed: %s", tc.desc, err))

		event := events.Event{
			Action: events.ThingGroupAndProfileUpdated{
				ID:        thingID,
				ProfileID: profileID,
				GroupID:   tc.newGroupID,
			},
			GroupID: groupID,
		}
		err = handler.Handle(context.Background(), event)
		require.Nil(t, err, fmt.Sprintf("%s: handling event failed: %s", tc.desc, err))

		page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
		require.Nil(t, err, fmt.Sprintf("%s: listing thing updates failed: %s", tc.desc, err))
		assert.Len(t, page.ThingUpdates, tc.size, fmt.Sprintf("%s: thing updates size mismatch", tc.desc))
	}
}
