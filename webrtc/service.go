// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webrtc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

const (
	protocol = "webrtc"

	subtopicOffer     = "webrtc.offer"
	subtopicAnswer    = "webrtc.answer"
	subtopicCandidate = "webrtc.candidate"
	subtopicConfig    = "webrtc.config"
)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ViewICEServers returns the ICE servers, with short-lived TURN credentials
	// when configured, for the user identified by the provided token.
	ViewICEServers(ctx context.Context, token string) ([]ICEServer, error)

	// View attaches the user to the live stream of the thing identified by the
	// provided ID and returns the viewer session with the SDP answer.
	View(ctx context.Context, token, thingID, offer string) (Session, error)

	// AddViewerCandidate adds a remote ICE candidate to the viewer session.
	AddViewerCandidate(ctx context.Context, token, sessionID string, candidate ICECandidate) error

	// ConsumeMessage handles the device signaling messages published
	// on the reserved webrtc subjects.
	ConsumeMessage(subject string, msg protomfx.Message) error
}

type webrtcService struct {
	things     domain.ThingsClient
	auth       domain.AuthClient
	sfu        SFU
	commands   messaging.CommandPublisher
	idProvider uuid.IDProvider
	cfg        Config
}

var _ Service = (*webrtcService)(nil)

// New instantiates the webrtc service implementation.
func New(things domain.ThingsClient, auth domain.AuthClient, sfu SFU, commands messaging.CommandPublisher, idp uuid.IDProvider, cfg Config) Service {
	return &webrtcService{
		things:     things,
		auth:       auth,
		sfu:        sfu,
		commands:   commands,
		idProvider: idp,
		cfg:        cfg,
	}
}

func (ws *webrtcService) ViewICEServers(ctx context.Context, token string) ([]ICEServer, error) {
	identity, err := ws.auth.Identify(ctx, token)
	if err != nil {
		return nil, err
	}

	return iceServers(ws.cfg, identity.ID, time.Now()), nil
}

func (ws *webrtcService) View(ctx context.Context, token, thingID, offer string) (Session, error) {
	identity, err := ws.auth.Identify(ctx, token)
	if err != nil {
		return Session{}, err
	}

	if err := ws.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer}); err != nil {
		return Session{}, err
	}

	id, err := ws.idProvider.ID()
	if err != nil {
		return Session{}, err
	}

	answer, err := ws.sfu.View(id, thingID, identity.ID, offer)
	if err != nil {
		return Session{}, err
	}

	return Session{ID: id, SDP: answer}, nil
}

func (ws *webrtcService) ConsumeMessage(_ string, msg protomfx.Message) error {
	if len(msg.Records) == 0 {
		return errors.ErrMalformedEntity
	}
	payload := msg.Records[0].Payload

	switch msg.Subtopic {
	case subtopicOffer:
		return ws.consumeOffer(msg.ThingId, payload)
	case subtopicCandidate:
		return ws.consumeCandidate(msg.ThingId, payload)
	case subtopicConfig:
		return ws.publishCommand(msg.ThingId, subtopicConfig, map[string]any{
			"ice_servers": iceServers(ws.cfg, msg.ThingId, time.Now()),
		})
	default:
		return errors.ErrMalformedEntity
	}
}

func (ws *webrtcService) consumeOffer(thingID string, payload []byte) (err error) {
	var req struct {
		SDP string `json:"sdp"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.SDP == "" {
		return errors.ErrMalformedEntity
	}

	id, err := ws.idProvider.ID()
	if err != nil {
		return err
	}

	answered := make(chan struct{})
	defer close(answered)

	onCandidate := func(c ICECandidate) {
		<-answered
		if err != nil {
			return
		}
		_ = ws.publishCommand(thingID, subtopicCandidate, map[string]any{
			"session_id": id,
			"candidate":  c,
		})
	}

	answer, err := ws.sfu.Publish(id, thingID, req.SDP, onCandidate)
	if err != nil {
		return err
	}

	return ws.publishCommand(thingID, subtopicAnswer, map[string]any{
		"session_id":  id,
		"sdp":         answer,
		"ice_servers": iceServers(ws.cfg, thingID, time.Now()),
	})
}

func (ws *webrtcService) consumeCandidate(thingID string, payload []byte) error {
	var req struct {
		SessionID string       `json:"session_id"`
		Candidate ICECandidate `json:"candidate"`
	}
	if err := json.Unmarshal(payload, &req); err != nil || req.Candidate.Candidate == "" {
		return errors.ErrMalformedEntity
	}

	return ws.sfu.AddCandidate(req.SessionID, thingID, req.Candidate)
}

func (ws *webrtcService) publishCommand(thingID, subtopic string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return ws.commands.PublishCommand(nats.GetThingCommandsSubject(thingID, subtopic), protomfx.Command{
		RecipientId: thingID,
		Subtopic:    subtopic,
		Payload:     payload,
		Protocol:    protocol,
		Created:     time.Now().UnixNano(),
	})
}

func (ws *webrtcService) AddViewerCandidate(ctx context.Context, token, sessionID string, candidate ICECandidate) error {
	identity, err := ws.auth.Identify(ctx, token)
	if err != nil {
		return err
	}

	return ws.sfu.AddCandidate(sessionID, identity.ID, candidate)
}
