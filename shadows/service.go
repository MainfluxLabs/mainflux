// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package shadows

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	shadowProtocol = "shadow"
	shadowSubtopic = "shadow"
)

// Service specifies the API offered by the shadows service. All methods that
// accept a token use it to identify and authorize the user.
type Service interface {
	// UpdateDesiredState replaces the thing's desired state with the provided
	// document, pushes the resulting delta to the device, and returns the
	// updated shadow.
	UpdateDesiredState(ctx context.Context, token, thingID string, desired State) (Shadow, error)

	// ViewShadow returns the thing's shadow with its delta populated.
	ViewShadow(ctx context.Context, token, thingID string) (Shadow, error)

	// RemoveShadow removes the thing's shadow.
	RemoveShadow(ctx context.Context, token, thingID string) error

	// RemoveByThing removes the shadow for the given thing without an auth
	// check. It is driven by thing-deleted events, not user requests.
	RemoveByThing(ctx context.Context, thingID string) error

	consumers.MessageConsumer
}

type shadowsService struct {
	things    domain.ThingsClient
	shadows   ShadowRepository
	publisher messaging.CommandPublisher
	logger    logger.Logger
}

var _ Service = (*shadowsService)(nil)

// New instantiates the shadows service.
func New(things domain.ThingsClient, shadows ShadowRepository, pub messaging.CommandPublisher, logger logger.Logger) Service {
	return &shadowsService{
		things:    things,
		shadows:   shadows,
		publisher: pub,
		logger:    logger,
	}
}

func (ss *shadowsService) UpdateDesiredState(ctx context.Context, token, thingID string, desired State) (Shadow, error) {
	if err := ss.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupEditor}); err != nil {
		return Shadow{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	current, err := ss.shadows.RetrieveByThing(ctx, thingID)
	if err != nil {
		return Shadow{}, err
	}

	stored, err := ss.shadows.Upsert(ctx, Shadow{
		ThingID:   thingID,
		Desired:   desired,
		Reported:  current.Reported,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return Shadow{}, err
	}

	stored.Delta = computeDelta(stored.Desired, stored.Reported)
	if err := ss.publishDelta(thingID, stored.Delta); err != nil {
		ss.logger.Warn(fmt.Sprintf("failed to push delta to thing %s: %s", thingID, err))
	}

	return stored, nil
}

func (ss *shadowsService) ViewShadow(ctx context.Context, token, thingID string) (Shadow, error) {
	if err := ss.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer}); err != nil {
		return Shadow{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	shadow, err := ss.shadows.RetrieveByThing(ctx, thingID)
	if err != nil {
		return Shadow{}, err
	}
	shadow.Delta = computeDelta(shadow.Desired, shadow.Reported)

	return shadow, nil
}

func (ss *shadowsService) RemoveShadow(ctx context.Context, token, thingID string) error {
	if err := ss.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupEditor}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return ss.shadows.Remove(ctx, thingID)
}

func (ss *shadowsService) RemoveByThing(ctx context.Context, thingID string) error {
	return ss.shadows.Remove(ctx, thingID)
}

// ConsumeMessage merges a thing's telemetry into its reported state. It skips
// no-op updates entirely, and re-publishes the delta only when one remains
func (ss *shadowsService) ConsumeMessage(_ string, msg protomfx.Message) error {
	thingID := msg.Publisher
	if thingID == "" {
		return nil
	}

	patch, ok := decodeState(msg)
	if !ok || len(patch) == 0 {
		return nil
	}

	ctx := context.Background()
	current, err := ss.shadows.RetrieveByThing(ctx, thingID)
	if err != nil {
		return err
	}

	merged, changed := mergeState(current.Reported, patch)
	if !changed {
		return nil
	}

	stored, err := ss.shadows.Upsert(ctx, Shadow{
		ThingID:   thingID,
		Desired:   current.Desired,
		Reported:  merged,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		return err
	}

	if delta := computeDelta(stored.Desired, stored.Reported); len(delta) > 0 {
		if err := ss.publishDelta(thingID, delta); err != nil {
			ss.logger.Warn(fmt.Sprintf("failed to push delta to thing %s: %s", thingID, err))
		}
	}

	return nil
}

// publishDelta publishes the delta to the thing's command subject.
// An empty delta is not published.
func (ss *shadowsService) publishDelta(thingID string, delta State) error {
	if len(delta) == 0 {
		return nil
	}

	payload, err := json.Marshal(delta)
	if err != nil {
		return err
	}

	cmd := protomfx.Command{
		Publisher: thingID,
		Subtopic:  shadowSubtopic,
		Protocol:  shadowProtocol,
		Payload:   payload,
		Created:   time.Now().UnixNano(),
	}

	subject := nats.GetThingCommandsSubject(thingID, shadowSubtopic)
	return ss.publisher.PublishCommand(subject, cmd)
}

// decodeState interprets a message payload as a flat state patch. SenML
// payloads are flattened to one key per record (keyed by record name); JSON
// object payloads are used as-is. Payloads that match neither shape are skipped.
func decodeState(msg protomfx.Message) (State, bool) {
	if msg.ContentType == messaging.SenMLContentType {
		return decodeSenML(msg)
	}

	var s State
	if err := json.Unmarshal(msg.Payload, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeSenML(msg protomfx.Message) (State, bool) {
	records, err := messaging.SplitMessage(msg)
	if err != nil {
		return nil, false
	}

	s := State{}
	for _, r := range records {
		sm, err := messaging.ToSenMLMessage(r)
		if err != nil {
			continue
		}

		switch {
		case sm.Value != nil:
			s[sm.Name] = *sm.Value
		case sm.StringValue != nil:
			s[sm.Name] = *sm.StringValue
		case sm.BoolValue != nil:
			s[sm.Name] = *sm.BoolValue
		case sm.DataValue != nil:
			s[sm.Name] = *sm.DataValue
		}
	}

	return s, len(s) > 0
}
