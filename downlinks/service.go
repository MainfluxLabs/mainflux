// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	clientshttp "github.com/MainfluxLabs/mainflux/pkg/clients/http"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/protoutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"golang.org/x/time/rate"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateDownlinks creates downlinks for certain thing identified by the thing ID.
	CreateDownlinks(ctx context.Context, token, thingID string, Downlinks ...Downlink) ([]Downlink, error)

	// ListDownlinksByThing retrieves data about a subset of downlinks
	// related to a certain thing.
	ListDownlinksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (DownlinksPage, error)

	// ListDownlinksByGroup retrieves data about a subset of downlinks
	// related to a certain group.
	ListDownlinksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (DownlinksPage, error)

	// ViewDownlink retrieves data about the downlink identified with the provided ID.
	ViewDownlink(ctx context.Context, token, id string) (Downlink, error)

	// UpdateDownlink updates the downlink identified by the provided ID.
	UpdateDownlink(ctx context.Context, token string, downlink Downlink) error

	// RemoveDownlinks removes downlinks identified with the provided IDs.
	RemoveDownlinks(ctx context.Context, token string, id ...string) error

	// RemoveDownlinksByThing removes downlinks related to the specified thing,
	// identified by the provided thing ID.
	RemoveDownlinksByThing(ctx context.Context, thingID string) error

	// RemoveDownlinksByGroup removes downlinks related to the specified group,
	// identified by the provided group ID.
	RemoveDownlinksByGroup(ctx context.Context, groupID string) error

	// RescheduleTasks reschedules all tasks for things associated with the specified profile ID.
	RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error

	// LoadAndScheduleTasks loads schedulers and starts them for executing downlinks
	LoadAndScheduleTasks(ctx context.Context) error

	// Backup retrieves all downlinks for backup purposes.
	Backup(ctx context.Context, token string) ([]Downlink, error)

	// Restore saves downlinks from a backup.
	Restore(ctx context.Context, token string, downlinks []Downlink) error
}

type downlinksService struct {
	things     protomfx.ThingsServiceClient
	auth       protomfx.AuthServiceClient
	downlinks  DownlinkRepository
	idProvider uuid.IDProvider
	publisher  messaging.Publisher
	logger     logger.Logger
	scheduler  *cron.ScheduleManager
	limiters   map[string]*rate.Limiter
	limiterMux sync.Mutex
}

const (
	downlinkProtocol = "http-downlink"
	taskTimeout      = 30 * time.Second

	MinuteInterval = "minute"
	HourInterval   = "hour"
	DayInterval    = "day"
)

var (
	errRetrieveHTTPResponse = "failed to retrieve HTTP response"
	errParsePayload         = "failed to parse payload"
	errPublishMessage       = "failed to publish a message"
	errFormatURL            = "failed to format URL"
	errExtractBaseURL       = "failed to extract base URL"
	errRateLimiter          = "failed to wait for rate limiter"
)

var _ Service = (*downlinksService)(nil)

func New(things protomfx.ThingsServiceClient, auth protomfx.AuthServiceClient, pub messaging.Publisher, downlinks DownlinkRepository, idp uuid.IDProvider, logger logger.Logger) Service {
	return &downlinksService{
		things:     things,
		auth:       auth,
		publisher:  pub,
		downlinks:  downlinks,
		idProvider: idp,
		logger:     logger,
		scheduler:  cron.NewScheduleManager(),
		limiters:   make(map[string]*rate.Limiter),
	}
}

func (ds *downlinksService) CreateDownlinks(ctx context.Context, token, thingID string, downlinks ...Downlink) ([]Downlink, error) {
	if _, err := ds.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	grID, err := ds.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return []Downlink{}, err
	}
	groupID := grID.GetValue()

	for i := range downlinks {
		downlinks[i].ThingID = thingID
		downlinks[i].GroupID = groupID

		id, err := ds.idProvider.ID()
		if err != nil {
			return []Downlink{}, err
		}
		downlinks[i].ID = id
	}

	dls, err := ds.downlinks.Save(ctx, downlinks...)
	if err != nil {
		return []Downlink{}, err
	}

	if err := ds.scheduleTasks(ctx, dls...); err != nil {
		return nil, err
	}

	return dls, nil
}

func (ds *downlinksService) ListDownlinksByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (DownlinksPage, error) {
	if _, err := ds.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer}); err != nil {
		return DownlinksPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	downlinks, err := ds.downlinks.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return DownlinksPage{}, err
	}

	return downlinks, nil
}

func (ds *downlinksService) ListDownlinksByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (DownlinksPage, error) {
	if _, err := ds.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
		return DownlinksPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	downlinks, err := ds.downlinks.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return DownlinksPage{}, err
	}

	return downlinks, nil
}

func (ds *downlinksService) ViewDownlink(ctx context.Context, token, id string) (Downlink, error) {
	downlink, err := ds.downlinks.RetrieveByID(ctx, id)
	if err != nil {
		return Downlink{}, err
	}

	if _, err := ds.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: downlink.ThingID, Action: things.Viewer}); err != nil {
		return Downlink{}, err
	}

	return downlink, nil
}

func (ds *downlinksService) UpdateDownlink(ctx context.Context, token string, downlink Downlink) error {
	dl, err := ds.downlinks.RetrieveByID(ctx, downlink.ID)
	if err != nil {
		return err
	}

	if _, err := ds.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: dl.ThingID, Action: things.Editor}); err != nil {
		return err
	}

	ds.unscheduleTask(dl)

	if err = ds.downlinks.Update(ctx, downlink); err != nil {
		return err
	}

	downlink.ThingID = dl.ThingID
	return ds.scheduleTasks(ctx, downlink)
}

func (ds *downlinksService) RemoveDownlinks(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		downlink, err := ds.downlinks.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if _, err := ds.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: downlink.ThingID, Action: things.Editor}); err != nil {
			return err
		}

		ds.unscheduleTask(downlink)
	}

	return ds.downlinks.Remove(ctx, ids...)
}

func (ds *downlinksService) RemoveDownlinksByThing(ctx context.Context, thingID string) error {
	page, err := ds.downlinks.RetrieveByThing(ctx, thingID, apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	if len(page.Downlinks) == 0 {
		return nil
	}

	for _, c := range page.Downlinks {
		ds.unscheduleTask(c)
	}

	return ds.downlinks.RemoveByThing(ctx, thingID)
}

func (ds *downlinksService) RemoveDownlinksByGroup(ctx context.Context, groupID string) error {
	page, err := ds.downlinks.RetrieveByGroup(ctx, groupID, apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	if len(page.Downlinks) == 0 {
		return nil
	}

	for _, d := range page.Downlinks {
		ds.unscheduleTask(d)
	}

	return ds.downlinks.RemoveByGroup(ctx, groupID)
}

func (ds *downlinksService) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error {
	var downlinks []Downlink

	thingIDs, err := ds.things.GetThingIDsByProfile(ctx, &protomfx.ProfileID{Value: profileID})
	if err != nil {
		return err
	}

	for _, thingID := range thingIDs.GetIds() {
		page, err := ds.downlinks.RetrieveByThing(ctx, thingID, apiutil.PageMetadata{})
		if err != nil {
			return err
		}
		downlinks = append(downlinks, page.Downlinks...)
	}

	if len(downlinks) == 0 {
		return nil
	}

	cfg := protoutil.MapToProtoConfig(config)

	for _, d := range downlinks {
		ds.unscheduleTask(d)

		if err := ds.scheduleTask(d, cfg); err != nil {
			return err
		}
	}

	ds.logger.Info(fmt.Sprintf("rescheduled %d tasks for profile %s", len(downlinks), profileID))

	return nil
}

func (ds *downlinksService) scheduleTasks(ctx context.Context, dls ...Downlink) error {
	for _, d := range dls {
		c, err := ds.things.GetConfigByThing(ctx, &protomfx.ThingID{Value: d.ThingID})
		if err != nil {
			return err
		}

		if err := ds.scheduleTask(d, c.GetConfig()); err != nil {
			return err
		}
	}

	return nil
}

func (ds *downlinksService) scheduleTask(d Downlink, cfg *protomfx.Config) error {
	task := ds.createTask(d, cfg)

	if d.Scheduler.Frequency != cron.OnceFreq {
		return ds.scheduler.ScheduleRepeatingTask(task, d.Scheduler, d.ID)
	}

	return ds.scheduler.ScheduleOneTimeTask(task, d.Scheduler, d.ID)
}

func (ds *downlinksService) unscheduleTask(d Downlink) {
	if d.Scheduler.Frequency != cron.OnceFreq {
		ds.scheduler.RemoveCronEntry(d.ID, d.Scheduler.TimeZone)
	}

	if t, ok := ds.scheduler.TimerByID[d.ID]; ok {
		t.Stop()
		delete(ds.scheduler.TimerByID, d.ID)
	}
}

func (ds *downlinksService) createTask(d Downlink, config *protomfx.Config) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
		defer cancel()

		path := d.Url
		if d.TimeFilter.StartParam != "" && d.TimeFilter.EndParam != "" {
			formattedURL, err := formatURL(d)
			if err != nil {
				ds.logger.Error(fmt.Sprintf("%s: %s", errFormatURL, err))
				return
			}
			path = formattedURL
		}

		baseURL, err := getBaseURL(path)
		if err != nil {
			ds.logger.Error(fmt.Sprintf("%s: %s", errExtractBaseURL, err))
			return
		}

		limiter := ds.getLimiter(baseURL)
		if err := limiter.Wait(ctx); err != nil {
			ds.logger.Error(fmt.Sprintf("%s: %s", errRateLimiter, err))
			return
		}

		response, err := clientshttp.SendRequest(d.Method, path, d.Payload, d.Headers)
		if err != nil {
			ds.logger.Error(fmt.Sprintf("%s: %s", errRetrieveHTTPResponse, err))
			return
		}

		formattedPayload, err := formatPayload(response)
		if err != nil {
			ds.logger.Error(fmt.Sprintf("%s: %s", errParsePayload, err))
			return
		}

		if err := ds.publish(config, d.ThingID, formattedPayload); err != nil {
			ds.logger.Error(fmt.Sprintf("%s with publisher %s: %s", errPublishMessage, d.ThingID, err))
			return
		}

		ds.logger.Info(fmt.Sprintf("task executed for downlink %s, thing %s", d.ID, d.ThingID))
	}
}

func (ds *downlinksService) LoadAndScheduleTasks(ctx context.Context) error {
	var dls []Downlink
	downlinks, err := ds.downlinks.RetrieveAll(ctx)
	if err != nil {
		return err
	}

	for _, d := range downlinks {
		if d.Scheduler.Frequency != cron.OnceFreq {
			dls = append(dls, d)
			continue
		}

		scheduledDateTime, err := cron.ParseTime(cron.DateTimeLayout, d.Scheduler.DateTime, d.Scheduler.TimeZone)
		if err != nil {
			return err
		}

		now := time.Now().In(scheduledDateTime.Location())
		if !scheduledDateTime.After(now) {
			ds.logger.Info(fmt.Sprintf("skipping past one-time downlink %s scheduled at %s", d.ID, scheduledDateTime))
			continue
		}
		dls = append(dls, d)
	}

	if err := ds.scheduleTasks(ctx, dls...); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		ds.scheduler.Stop()
	}()

	return nil
}

func (ds *downlinksService) publish(config *protomfx.Config, thingID string, payload []byte) error {
	msg := protomfx.Message{
		Protocol: downlinkProtocol,
		Payload:  payload,
	}

	conn := &protomfx.PubConfigByKeyRes{PublisherID: thingID, ProfileConfig: config}
	if err := messaging.FormatMessage(conn, &msg); err != nil {
		return err
	}

	msg.Subject = nats.GetMessagesSubject(msg.Publisher, msg.Subtopic)

	return ds.publisher.Publish(msg)
}

func (ds *downlinksService) getLimiter(baseURL string) *rate.Limiter {
	ds.limiterMux.Lock()
	defer ds.limiterMux.Unlock()

	if limiter, ok := ds.limiters[baseURL]; ok {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Every(2500*time.Millisecond), 1)
	ds.limiters[baseURL] = limiter
	return limiter
}

func (ds *downlinksService) Backup(ctx context.Context, token string) ([]Downlink, error) {
	if err := ds.isAdmin(ctx, token); err != nil {
		return nil, err
	}

	return ds.downlinks.RetrieveAll(ctx)
}

func (ds *downlinksService) Restore(ctx context.Context, token string, dls []Downlink) error {
	if err := ds.isAdmin(ctx, token); err != nil {
		return err
	}

	if _, err := ds.downlinks.Save(ctx, dls...); err != nil {
		return err
	}

	return ds.scheduleTasks(ctx, dls...)
}

func (ds *downlinksService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: "root",
	}

	if _, err := ds.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
