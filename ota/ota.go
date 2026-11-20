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

const (
	// UpdateStatusInProgress means at least one thing is still active.
	UpdateStatusInProgress = "in_progress"
	// UpdateStatusCompleted means no thing is active or failed.
	UpdateStatusCompleted = "completed"
	// UpdateStatusCompletedWithFailures means no thing is active and at least one failed.
	UpdateStatusCompletedWithFailures = "completed_with_failures"
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
	Status      string
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

// Status returns the update status the stats add up to.
func (s UpdateStats) Status() string {
	switch {
	case s.Pending+s.Downloading+s.Installing > 0:
		return UpdateStatusInProgress
	case s.Failed > 0:
		return UpdateStatusCompletedWithFailures
	default:
		return UpdateStatusCompleted
	}
}

// Add counts a thing with the given status.
func (s *UpdateStats) Add(status string) {
	s.Targets++
	if counter := s.counter(status); counter != nil {
		*counter++
	}
}

// Remove stops counting a thing with the given status.
func (s *UpdateStats) Remove(status string) {
	s.Targets--
	if counter := s.counter(status); counter != nil {
		*counter--
	}
}

func (s *UpdateStats) counter(status string) *uint64 {
	switch status {
	case StatusPending:
		return &s.Pending
	case StatusDownloading:
		return &s.Downloading
	case StatusInstalling:
		return &s.Installing
	case StatusSucceeded:
		return &s.Succeeded
	case StatusFailed:
		return &s.Failed
	default:
		return nil
	}
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
	"id":          "id",
	"created_at":  "created_at",
	"version":     "version",
	"status":      "status",
	"targets":     "targets",
	"pending":     "pending_count",
	"downloading": "downloading_count",
	"installing":  "installing_count",
	"succeeded":   "succeeded_count",
	"failed":      "failed_count",
}

// ThingUpdateOrderFields maps API order keys to SQL column expressions of the thing_updates table.
var ThingUpdateOrderFields = map[string]string{
	"thing_id":   "thing_id",
	"updated_at": "updated_at",
	"status":     "status",
}

// UpdateRepository specifies the persistence API for updates and their thing updates.
type UpdateRepository interface {
	// Save persists the update with a pending thing update for each given thing. Older active
	// thing updates of the same things are marked failed as superseded in the same transaction.
	Save(ctx context.Context, u Update, thingIDs ...string) error

	// RetrieveByID retrieves the update with the given ID, including its status and stats.
	RetrieveByID(ctx context.Context, id string) (Update, error)

	// RetrieveByGroup retrieves a page of updates of the given group, including their status and
	// stats, optionally filtered by status.
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

	// UpdateThingUpdate updates the progress of an active thing update.
	UpdateThingUpdate(ctx context.Context, tu ThingUpdate) error

	// RetrieveFailedThingIDs retrieves the IDs of the failed things of the given update.
	RetrieveFailedThingIDs(ctx context.Context, updateID string) ([]string, error)

	// RetryFailed moves the failed thing updates of the given update and things back to pending
	// and returns the IDs of the affected things.
	RetryFailed(ctx context.Context, updateID string, thingIDs []string) ([]string, error)

	// FailTimedOut fails active thing updates without progress for their update's timeout.
	FailTimedOut(ctx context.Context, now time.Time) error

	// RemoveThingUpdatesByThing removes all thing updates of the given thing.
	RemoveThingUpdatesByThing(ctx context.Context, thingID string) error
}
