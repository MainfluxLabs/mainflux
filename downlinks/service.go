// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package downlinks

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sync"

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
	downlinkProtocol     = "http-downlink"
	baseISO8601Format    = "2006-01-02T15:04:05"
	compactISO8601Format = "200601021504"
	contentType          = "Content-Type"
	jsonFormat           = "json"
	xmlFormat            = "xml"

	MinuteInterval = "minute"
	HourInterval   = "hour"
	DayInterval    = "day"
)

var _ Service = (*downlinksService)(nil)
var (
	errRetrieveHTTPResponse = "failed to retrieve HTTP response"
	errParsePayload         = "failed to parse payload"
	errPublishMessage       = "failed to publish a message"
	errFormatURL            = "failed to format URL"
	errExtractBaseURL       = "failed to extract base URL"
	errRateLimiter          = "failed to wait for rate limiter"
)

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
	if err := ds.scheduleTasks(ctx, downlink); err != nil {
		return err
	}

	return nil
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

	if err := ds.downlinks.Remove(ctx, ids...); err != nil {
		return err
	}

	return nil
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

	// stop existing tasks and start new tasks with updated config
	for _, d := range downlinks {
		ds.unscheduleTask(d)

		if err := ds.scheduleTask(d, cfg); err != nil {
			return err
		}
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
			ds.logger.Error(fmt.Sprintf("%s: %s", errPublishMessage, err))
		}
	}
}

func (ds *downlinksService) LoadAndScheduleTasks(ctx context.Context) error {
	var dls []Downlink
	downlinks, err := ds.downlinks.RetrieveAll(ctx)
	if err != nil {
		return err
	}

	for _, d := range downlinks {
		if d.Scheduler.Frequency == cron.OnceFreq {
			scheduledDateTime, err := cron.ParseTime(cron.DateTimeLayout, d.Scheduler.DateTime, d.Scheduler.TimeZone)
			if err != nil {
				return err
			}

			now := time.Now().In(scheduledDateTime.Location())
			if scheduledDateTime.After(now) {
				dls = append(dls, d)
			}
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

	if err := ds.publisher.Publish(msg); err != nil {
		return err
	}

	return nil
}

func formatTime(t time.Time, format string) string {
	switch format {
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "unix_ms":
		return fmt.Sprintf("%d", t.UnixNano()/1e6)
	case "unix_us":
		return fmt.Sprintf("%d", t.UnixNano()/1e3)
	case "unix_ns":
		return fmt.Sprintf("%d", t.UnixNano())
	default:
		layout := getLayout(format)
		return t.Format(layout)
	}
}

func getLayout(format string) string {
	switch strings.ToLower(format) {
	case "ansic":
		return time.ANSIC
	case "unixdate":
		return time.UnixDate
	case "rubydate":
		return time.RubyDate
	case "rfc822":
		return time.RFC822
	case "rfc822z":
		return time.RFC822Z
	case "rfc850":
		return time.RFC850
	case "rfc1123":
		return time.RFC1123
	case "rfc1123z":
		return time.RFC1123Z
	case "rfc3339":
		return time.RFC3339
	case "rfc3339nano":
		return time.RFC3339Nano
	case "stamp":
		return time.Stamp
	case "stampmilli":
		return time.StampMilli
	case "stampmicro":
		return time.StampMicro
	case "stampnano":
		return time.StampNano
	case "iso8601":
		return baseISO8601Format
	case "datetime":
		return time.DateTime
	case "compactiso8601":
		return compactISO8601Format
	}

	return ""
}

func formatURL(d Downlink) (string, error) {
	u, err := url.Parse(d.Url)
	if err != nil {
		return "", err
	}

	startTime, endTime, err := calculateTimeRange(d.Scheduler.TimeZone, d.TimeFilter)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set(d.TimeFilter.StartParam, formatTime(startTime, d.TimeFilter.Format))
	q.Set(d.TimeFilter.EndParam, formatTime(endTime, d.TimeFilter.Format))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func calculateTimeRange(timezone string, filter TimeFilter) (time.Time, time.Time, error) {
	var duration time.Duration

	switch filter.Interval {
	case MinuteInterval:
		duration = time.Duration(filter.Value) * time.Minute
	case HourInterval:
		duration = time.Duration(filter.Value) * time.Hour
	case DayInterval:
		duration = time.Duration(filter.Value*24) * time.Hour
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	now := time.Now().In(loc)

	if filter.Forecast {
		return now, now.Add(duration), nil
	}
	return now.Add(-duration), now, nil
}
 
func formatPayload(response *http.Response) ([]byte, error) {
	var mappedData = payload{}
	resPayload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	format := getFormat(response.Header.Get(contentType))

	switch format {
	case xmlFormat:
		err := xml.Unmarshal(resPayload, &mappedData)
		if err != nil {
			return nil, err
		}

		filteredData := removeUnderscoreKeys(mappedData.data)
		jsonData, err := json.Marshal(filteredData)
		if err != nil {
			return nil, err
		}

		return jsonData, nil
	case jsonFormat:
		return resPayload, nil
	default:
		// Build comprehensive error information
		errorInfo := map[string]any{
			"error":            string(resPayload),
			"http_status":      response.Status,
			"status_code":      response.StatusCode,
			"response_headers": response.Header,
			"request_method":   response.Request.Method,
			"request_url":      response.Request.URL.String(),
		}

		return json.Marshal(errorInfo)
	}
}

func (p *payload) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	p.data = make(map[string]any)
	stack := []map[string]any{p.data}
	nameStack := []xml.Name{start.Name}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Create new node
			node := map[string]any{}

			// Add to parent
			parent := stack[len(stack)-1]
			currentName := elem.Name.Local

			if existing, exists := parent[currentName]; exists {
				switch v := existing.(type) {
				case []any:
					parent[currentName] = append(v, node)
				case map[string]any:
					parent[currentName] = []any{v, node}
				}
			} else {
				parent[currentName] = node
			}

			stack = append(stack, node)
			nameStack = append(nameStack, elem.Name)

		case xml.EndElement:
			stack = stack[:len(stack)-1]
			nameStack = nameStack[:len(nameStack)-1]

		case xml.CharData:
			val := strings.TrimSpace(string(elem))
			if val == "" {
				continue
			}

			current := stack[len(stack)-1]
			if len(current) == 0 {
				// Simple text content
				parent := stack[len(stack)-2]
				name := nameStack[len(nameStack)-1].Local
				parent[name] = val
			} else {
				// Mixed content
				current["#text"] = val
			}
		}
	}
}

func getFormat(ct string) string {
	switch {
	case strings.Contains(ct, jsonFormat):
		return jsonFormat
	case strings.Contains(ct, xmlFormat):
		return xmlFormat
	default:
		return ""
	}
}

func removeUnderscoreKeys(data map[string]any) map[string]any {
	filteredData := make(map[string]any)

	for key, value := range data {
		if key == "_" {
			continue
		}

		v, ok := value.(map[string]any)
		if ok {
			filteredData[key] = removeUnderscoreKeys(v)
			continue
		}

		filteredData[key] = value
	}

	return filteredData
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

func getBaseURL(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path), nil
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

	return nil
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
