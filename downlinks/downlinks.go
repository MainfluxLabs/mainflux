// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
)

type Metadata map[string]any

type Downlink struct {
	ID         string
	GroupID    string
	ThingID    string
	Name       string
	Url        string
	Method     string
	Payload    []byte
	Headers    map[string]string
	Scheduler  cron.Scheduler
	TimeFilter TimeFilter
	Metadata   Metadata
}

type TimeFilter struct {
	StartParam string `json:"start_param"`
	EndParam   string `json:"end_param"`
	Format     string `json:"format"`
	Forecast   bool   `json:"forecast"`
	Interval   string `json:"interval"` // minute | hour | day
	Value      uint   `json:"value"`
}

type DownlinksPage struct {
	apiutil.PageMetadata
	Downlinks []Downlink
}

type DownlinkRepository interface {
	// Save persists multiple downlinks. Downlinks are saved using a transaction.
	// If one downlink fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, dls ...Downlink) ([]Downlink, error)

	// RetrieveByThing retrieves downlinks related to a certain thing,
	// identified by a given thing ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (DownlinksPage, error)

	// RetrieveByGroup retrieves downlinks related to a certain group,
	// identified by a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (DownlinksPage, error)

	// RetrieveByID retrieves the downlink having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Downlink, error)

	// RetrieveAll retrieves all downlinks.
	RetrieveAll(ctx context.Context) ([]Downlink, error)

	// Update performs an update to the existing downlink.
	// A non-nil error is returned to indicate operation failure.
	Update(ctx context.Context, d Downlink) error

	// Remove removes downlinks having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByThing removes downlinks related to a certain thing,
	// identified by a given thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes downlinks related to a certain group,
	// identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error
}
