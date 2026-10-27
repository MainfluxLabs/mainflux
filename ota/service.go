// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	defaultTimeout = 24 * time.Hour
	maxErrorLen    = 255
	commandSource  = "ota"
)

var (
	// ErrInvalidStatus indicates a status report with an unknown status.
	ErrInvalidStatus = errors.New("invalid ota status")

	// ErrUpdateFinished indicates a status report for a thing update that already succeeded or failed.
	ErrUpdateFinished = errors.New("ota update already finished")

	// ErrMissingReportedVersion indicates a succeeded status report without the installed version.
	ErrMissingReportedVersion = errors.New("missing reported version")
)

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total  uint64 `json:"total,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Limit  uint64 `json:"limit,omitempty"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Status string `json:"status,omitempty"`
}

// StatusReport is a thing's report on the progress of an update.
type StatusReport struct {
	UpdateID string `json:"update_id"`
	Status   string `json:"status"`
	Version  string `json:"version,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Command tells a thing which firmware to install and where to download it from.
type Command struct {
	UpdateID string `json:"update_id"`
	Version  string `json:"version"`
	FileName string `json:"file_name"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateUpdate creates an update of the given things in the group and notifies them.
	CreateUpdate(ctx context.Context, token, groupID string, u Update, thingIDs ...string) (Update, error)

	// ListUpdatesByGroup retrieves a page of updates of the given group.
	ListUpdatesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (UpdatesPage, error)

	// ViewUpdate retrieves the update identified by the given ID.
	ViewUpdate(ctx context.Context, token, id string) (Update, error)

	// ListThingUpdates retrieves a page of thing updates of the given update.
	ListThingUpdates(ctx context.Context, token, updateID string, pm PageMetadata) (ThingUpdatesPage, error)

	// RetryFailed moves the failed things of the given update back to pending and notifies them again.
	RetryFailed(ctx context.Context, token, updateID string) error

	// RemoveUpdate removes the update identified by the given ID.
	RemoveUpdate(ctx context.Context, token, id string) error

	// ViewPending retrieves the command for the update the thing identified by the key should apply.
	ViewPending(ctx context.Context, key domain.ThingKey) (Command, error)

	// ReportStatus records the progress of an update reported by the thing identified by the key.
	ReportStatus(ctx context.Context, key domain.ThingKey, report StatusReport) error

	// FailTimedOut marks thing updates without progress for their update's timeout as failed.
	FailTimedOut(ctx context.Context) error

	// RemoveThingUpdatesByThing removes thing updates of the given thing.
	RemoveThingUpdatesByThing(ctx context.Context, thingID string) error

	// RemoveUpdatesByGroup removes updates of the given group.
	RemoveUpdatesByGroup(ctx context.Context, groupID string) error

	consumers.MessageConsumer
}

type otaService struct {
	things      domain.ThingsClient
	updates     UpdateRepository
	publisher   messaging.CommandPublisher
	idProvider  uuid.IDProvider
	firmwareURL string
	logger      logger.Logger
}

var _ Service = (*otaService)(nil)

// New instantiates the ota service implementation.
// firmwareURL is the public base URL things download firmware files from.
func New(things domain.ThingsClient, updates UpdateRepository, publisher messaging.CommandPublisher, idp uuid.IDProvider, firmwareURL string, logger logger.Logger) Service {
	return &otaService{
		things:      things,
		updates:     updates,
		publisher:   publisher,
		idProvider:  idp,
		firmwareURL: strings.TrimSuffix(firmwareURL, "/"),
		logger:      logger,
	}
}

func (ots *otaService) CreateUpdate(ctx context.Context, token, groupID string, u Update, thingIDs ...string) (Update, error) {
	if len(thingIDs) == 0 {
		return Update{}, errors.ErrMalformedEntity
	}

	if err := ots.things.CanUserAccessThings(ctx, domain.UserAccessThingsReq{Token: token, IDs: thingIDs, GroupID: groupID, Action: domain.GroupEditor}); err != nil {
		return Update{}, err
	}

	id, err := ots.idProvider.ID()
	if err != nil {
		return Update{}, err
	}

	now := time.Now().UTC()
	u.ID = id
	u.GroupID = groupID
	u.CreatedAt = now
	if u.Timeout <= 0 {
		u.Timeout = defaultTimeout
	}

	if err := ots.updates.Save(ctx, u, thingIDs...); err != nil {
		return Update{}, err
	}

	ots.notify(u, thingIDs...)
	u.Stats = UpdateStats{Targets: uint64(len(thingIDs)), Pending: uint64(len(thingIDs))}

	return u, nil
}

func (ots *otaService) ListUpdatesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (UpdatesPage, error) {
	if err := ots.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return UpdatesPage{}, err
	}

	return ots.updates.RetrieveByGroup(ctx, groupID, pm)
}

func (ots *otaService) ViewUpdate(ctx context.Context, token, id string) (Update, error) {
	u, err := ots.updates.RetrieveByID(ctx, id)
	if err != nil {
		return Update{}, err
	}

	if err := ots.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: u.GroupID, Action: domain.GroupViewer}); err != nil {
		return Update{}, err
	}

	return u, nil
}

func (ots *otaService) ListThingUpdates(ctx context.Context, token, updateID string, pm PageMetadata) (ThingUpdatesPage, error) {
	if _, err := ots.ViewUpdate(ctx, token, updateID); err != nil {
		return ThingUpdatesPage{}, err
	}

	return ots.updates.RetrieveThingUpdates(ctx, updateID, pm)
}

func (ots *otaService) RetryFailed(ctx context.Context, token, updateID string) error {
	u, err := ots.updates.RetrieveByID(ctx, updateID)
	if err != nil {
		return err
	}

	if err := ots.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: u.GroupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	thingIDs, err := ots.updates.RetryFailed(ctx, updateID)
	if err != nil {
		return err
	}

	ots.notify(u, thingIDs...)

	return nil
}

func (ots *otaService) RemoveUpdate(ctx context.Context, token, id string) error {
	u, err := ots.updates.RetrieveByID(ctx, id)
	if err != nil {
		return err
	}

	if err := ots.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: u.GroupID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	return ots.updates.Remove(ctx, id)
}

func (ots *otaService) ViewPending(ctx context.Context, key domain.ThingKey) (Command, error) {
	thingID, err := ots.things.Identify(ctx, key)
	if err != nil {
		return Command{}, err
	}

	tu, err := ots.updates.RetrieveActiveThingUpdate(ctx, thingID)
	if err != nil {
		return Command{}, err
	}

	u, err := ots.updates.RetrieveByID(ctx, tu.UpdateID)
	if err != nil {
		return Command{}, err
	}

	return ots.newCommand(u), nil
}

func (ots *otaService) ReportStatus(ctx context.Context, key domain.ThingKey, report StatusReport) error {
	thingID, err := ots.things.Identify(ctx, key)
	if err != nil {
		return err
	}

	return ots.reportStatus(ctx, thingID, report)
}

func (ots *otaService) FailTimedOut(ctx context.Context) error {
	return ots.updates.FailTimedOut(ctx, time.Now().UTC())
}

func (ots *otaService) RemoveThingUpdatesByThing(ctx context.Context, thingID string) error {
	return ots.updates.RemoveThingUpdatesByThing(ctx, thingID)
}

func (ots *otaService) RemoveUpdatesByGroup(ctx context.Context, groupID string) error {
	return ots.updates.RemoveByGroup(ctx, groupID)
}

func (ots *otaService) ConsumeMessage(_ string, msg protomfx.Message) error {
	var report StatusReport
	if err := json.Unmarshal(msg.Payload, &report); err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return ots.reportStatus(context.Background(), msg.ThingId, report)
}

func (ots *otaService) reportStatus(ctx context.Context, thingID string, report StatusReport) error {
	switch report.Status {
	case StatusDownloading, StatusInstalling, StatusSucceeded, StatusFailed:
	default:
		return ErrInvalidStatus
	}

	if report.Status == StatusSucceeded && report.Version == "" {
		return ErrMissingReportedVersion
	}

	tu, err := ots.updates.RetrieveThingUpdate(ctx, report.UpdateID, thingID)
	if err != nil {
		return err
	}

	if tu.Status == StatusSucceeded || tu.Status == StatusFailed {
		return ErrUpdateFinished
	}

	tu.Status = report.Status
	tu.Error = truncate(report.Error, maxErrorLen)
	tu.UpdatedAt = time.Now().UTC()
	if report.Version != "" {
		tu.ReportedVersion = report.Version
	}

	if report.Status == StatusSucceeded {
		u, err := ots.updates.RetrieveByID(ctx, report.UpdateID)
		if err != nil {
			return err
		}

		if report.Version != u.Version {
			tu.Status = StatusFailed
			tu.Error = fmt.Sprintf(ErrMsgVersionMismatch, report.Version)
		}
	}

	return ots.updates.UpdateThingUpdate(ctx, tu)
}

// notify sends the ota command to the given things; offline things pick the update up through ViewPending.
func (ots *otaService) notify(u Update, thingIDs ...string) {
	payload, err := json.Marshal(ots.newCommand(u))
	if err != nil {
		ots.logger.Error(fmt.Sprintf("failed to encode ota command for update %s: %s", u.ID, err))
		return
	}

	for _, thingID := range thingIDs {
		cmd := protomfx.Command{
			RecipientId: thingID,
			Subtopic:    nats.SubtopicOTA,
			Payload:     payload,
			Protocol:    commandSource,
			Created:     time.Now().UnixNano(),
		}

		if err := ots.publisher.PublishCommand(nats.GetThingCommandsSubject(thingID, nats.SubtopicOTA), cmd); err != nil {
			ots.logger.Warn(fmt.Sprintf("failed to send ota command for update %s to thing %s: %s", u.ID, thingID, err))
		}
	}
}

func (ots *otaService) newCommand(u Update) Command {
	return Command{
		UpdateID: u.ID,
		Version:  u.Version,
		FileName: u.FileName,
		URL:      ots.firmwareURL + "/" + url.PathEscape(u.FileName),
		Size:     u.FileSize,
		Checksum: u.Checksum,
	}
}

// truncate shortens device-reported text so it fits the stored column.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}

	return string(runes[:max])
}
