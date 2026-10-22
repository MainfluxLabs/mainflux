// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ota

import (
	"context"
	"time"
)

const (
	// StatusPending means the thing has not started the update yet.
	StatusPending = "pending"
	// StatusDownloading means the thing is downloading the firmware.
	StatusDownloading = "downloading"
	// StatusInstalling means the thing is installing the firmware.
	StatusInstalling = "installing"
	// StatusSucceeded means the thing reported the target version.
	StatusSucceeded = "succeeded"
	// StatusFailed means the update failed, timed out or was superseded.
	StatusFailed = "failed"
)

const (
	// ErrMsgTimedOut is stored for a thing update without progress for its update's timeout.
	ErrMsgTimedOut = "timed out"
	// ErrMsgVersionMismatch is stored when a thing reports success with a different version.
	ErrMsgVersionMismatch = "version mismatch: reported %s"
	// ErrMsgSuperseded is stored for an active thing update replaced by a newer update.
	ErrMsgSuperseded = "superseded by update %s"
)

// ActiveStatuses are the statuses of a thing update that is still running.
var ActiveStatuses = []string{StatusPending, StatusDownloading, StatusInstalling}

// Update is a firmware rollout to a set of things in one group.
type Update struct {
	ID          string
	GroupID     string
	FileName    string
	Version     string
	FileSize    int64
	Checksum    string
	Description string
	Timeout     time.Duration
	CreatedAt   time.Time
	Stats       UpdateStats
}

// UpdateStats counts the things of an update by status.
type UpdateStats struct {
	Targets     uint64
	Pending     uint64
	Downloading uint64
	Installing  uint64
	Succeeded   uint64
	Failed      uint64
}

// ThingUpdate is the progress of one thing within an update.
type ThingUpdate struct {
	UpdateID        string
	ThingID         string
	Status          string
	Error           string
	ReportedVersion string
	UpdatedAt       time.Time
}

// UpdatesPage contains a page of updates.
type UpdatesPage struct {
	Total   uint64
	Updates []Update
}

// ThingUpdatesPage contains a page of thing updates.
type ThingUpdatesPage struct {
	Total        uint64
	ThingUpdates []ThingUpdate
}

// UpdateOrderFields maps API-facing order keys to SQL column expressions for the updates table.
var UpdateOrderFields = map[string]string{
	"id":         "id",
	"created_at": "created_at",
	"version":    "version",
}

// ThingUpdateOrderFields maps API-facing order keys to SQL column expressions for the thing updates table.
var ThingUpdateOrderFields = map[string]string{
	"id":         "thing_id",
	"updated_at": "updated_at",
	"status":     "status",
}

// UpdateRepository specifies the persistence API for updates and their thing updates.
type UpdateRepository interface {
	// Save persists the update with a pending thing update for each given thing. Older active
	// thing updates of the same things are marked failed as superseded in the same transaction.
	Save(ctx context.Context, u Update, thingIDs ...string) error

	// RetrieveByID retrieves the update with the given ID, including its stats.
	RetrieveByID(ctx context.Context, id string) (Update, error)

	// RetrieveByGroup retrieves a page of updates of the given group, including their stats.
	RetrieveByGroup(ctx context.Context, groupID string, pm PageMetadata) (UpdatesPage, error)

	// Remove removes the update with the given ID and its thing updates.
	Remove(ctx context.Context, id string) error

	// RemoveByGroup removes all updates of the given group.
	RemoveByGroup(ctx context.Context, groupID string) error

	// RetrieveThingUpdates retrieves a page of thing updates of the given update.
	RetrieveThingUpdates(ctx context.Context, updateID string, pm PageMetadata) (ThingUpdatesPage, error)

	// RetrieveThingUpdate retrieves the thing update of the given update and thing.
	RetrieveThingUpdate(ctx context.Context, updateID, thingID string) (ThingUpdate, error)

	// RetrieveActiveThingUpdate retrieves the newest active thing update of the given thing.
	RetrieveActiveThingUpdate(ctx context.Context, thingID string) (ThingUpdate, error)

	// UpdateThingUpdate updates the status, error, reported version and timestamps of a thing update.
	UpdateThingUpdate(ctx context.Context, tu ThingUpdate) error

	// RetryFailed moves the failed thing updates of the given update back to pending
	// and returns the IDs of the affected things.
	RetryFailed(ctx context.Context, updateID string) ([]string, error)

	// FailTimedOut marks active thing updates without progress for their update's timeout as failed.
	FailTimedOut(ctx context.Context, now time.Time) error

	// RemoveThingUpdatesByThing removes all thing updates of the given thing.
	RemoveThingUpdatesByThing(ctx context.Context, thingID string) error
}
