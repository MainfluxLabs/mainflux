// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webrtc_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/webrtc"
	wmocks "github.com/MainfluxLabs/mainflux/webrtc/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "user@example.com"
	otherToken   = "other@example.com"
	wrongValue   = "wrong-value"
	userID       = "574106f7-030e-4881-8ab0-151195c29f94"
	otherUserID  = "9b5a2cde-6b7e-4f2e-a6de-8f2f6a3a4a11"
	thingID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	otherThingID = "c3a1a6f0-3d1a-4a39-9f7b-5d3c5b5f0e22"
	publisherSID = "publisher-session"
	offer        = "v=0 offer"
	invalidSDP   = "invalid"
	stunURL      = "stun:stun.example.com:3478"
	turnURL      = "turn:turn.example.com:3478"
	turnSecret   = "turn-secret"
	turnTTL      = time.Hour
)

var candidate = webrtc.ICECandidate{Candidate: "candidate:1 1 UDP 1 127.0.0.1 5000 typ host"}

func newService(turn bool) (webrtc.Service, *wmocks.SFU, *wmocks.Publisher) {
	things := mocks.NewThingsServiceClient(nil, map[string]domain.Thing{
		token:      {ID: thingID},
		otherToken: {ID: otherThingID},
	}, nil)
	auth := mocks.NewAuthService("", []users.User{
		{ID: userID, Email: token},
		{ID: otherUserID, Email: otherToken},
	}, nil)
	sfu := wmocks.NewSFU()

	cfg := webrtc.Config{STUNURLs: []string{stunURL}}
	if turn {
		cfg.TURNURLs = []string{turnURL}
		cfg.TURNSecret = turnSecret
		cfg.TURNTTL = turnTTL
	}

	pub := wmocks.NewPublisher()

	return webrtc.New(things, auth, sfu, pub, uuid.NewMock(), cfg), sfu, pub
}

func TestViewICEServers(t *testing.T) {
	cases := []struct {
		desc  string
		token string
		turn  bool
		err   error
	}{
		{desc: "view ice servers with stun and turn", token: token, turn: true},
		{desc: "view ice servers without turn config", token: token},
		{desc: "view ice servers with invalid token", token: wrongValue, turn: true, err: errors.ErrAuthentication},
	}

	for _, tc := range cases {
		svc, _, _ := newService(tc.turn)
		now := time.Now()
		servers, err := svc.ViewICEServers(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err != nil {
			continue
		}

		require.NotEmpty(t, servers, fmt.Sprintf("%s: expected ice servers\n", tc.desc))
		assert.Equal(t, []string{stunURL}, servers[0].URLs, fmt.Sprintf("%s: unexpected stun urls\n", tc.desc))
		assert.Empty(t, servers[0].Username, fmt.Sprintf("%s: stun entry must not carry credentials\n", tc.desc))
		assert.Empty(t, servers[0].Credential, fmt.Sprintf("%s: stun entry must not carry credentials\n", tc.desc))

		if !tc.turn {
			assert.Len(t, servers, 1, fmt.Sprintf("%s: expected only the stun entry\n", tc.desc))
			continue
		}

		require.Len(t, servers, 2, fmt.Sprintf("%s: expected stun and turn entries\n", tc.desc))
		turn := servers[1]
		assert.Equal(t, []string{turnURL}, turn.URLs, fmt.Sprintf("%s: unexpected turn urls\n", tc.desc))

		parts := strings.SplitN(turn.Username, ":", 2)
		require.Len(t, parts, 2, fmt.Sprintf("%s: unexpected username %s\n", tc.desc, turn.Username))
		assert.Equal(t, userID, parts[1], fmt.Sprintf("%s: username must carry the user id\n", tc.desc))

		expiry, err := strconv.ParseInt(parts[0], 10, 64)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected expiry: %s\n", tc.desc, err))
		assert.GreaterOrEqual(t, expiry, now.Add(turnTTL-5*time.Second).Unix(), fmt.Sprintf("%s: expiry too early\n", tc.desc))
		assert.LessOrEqual(t, expiry, now.Add(turnTTL+5*time.Second).Unix(), fmt.Sprintf("%s: expiry too late\n", tc.desc))

		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(turn.Username))
		assert.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), turn.Credential, fmt.Sprintf("%s: unexpected credential\n", tc.desc))
	}
}

func TestView(t *testing.T) {
	cases := []struct {
		desc      string
		token     string
		thingID   string
		offer     string
		publisher bool
		err       error
	}{
		{desc: "view stream of accessible thing", token: token, thingID: thingID, offer: offer, publisher: true},
		{desc: "view stream with invalid token", token: wrongValue, thingID: thingID, offer: offer, publisher: true, err: errors.ErrAuthentication},
		{desc: "view stream of thing without access", token: token, thingID: otherThingID, offer: offer, publisher: true, err: errors.ErrAuthorization},
		{desc: "view stream of thing without publisher", token: token, thingID: thingID, offer: offer, err: dbutil.ErrNotFound},
		{desc: "view stream with malformed offer", token: token, thingID: thingID, offer: invalidSDP, publisher: true, err: errors.ErrMalformedEntity},
	}

	for _, tc := range cases {
		svc, sfu, _ := newService(false)
		if tc.publisher {
			_, err := sfu.Publish(publisherSID, thingID, offer, func(webrtc.ICECandidate) {})
			require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s\n", tc.desc, err))
		}

		session, err := svc.View(context.Background(), tc.token, tc.thingID, tc.offer)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.NotEmpty(t, session.ID, fmt.Sprintf("%s: expected session id\n", tc.desc))
			assert.NotEmpty(t, session.SDP, fmt.Sprintf("%s: expected answer sdp\n", tc.desc))
		}
	}
}

func TestAddViewerCandidate(t *testing.T) {
	svc, sfu, _ := newService(false)
	_, err := sfu.Publish(publisherSID, thingID, offer, func(webrtc.ICECandidate) {})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	session, err := svc.View(context.Background(), token, thingID, offer)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc      string
		token     string
		sessionID string
		err       error
	}{
		{desc: "add candidate to own session", token: token, sessionID: session.ID},
		{desc: "add candidate to unknown session", token: token, sessionID: wrongValue, err: dbutil.ErrNotFound},
		{desc: "add candidate to session of another user", token: otherToken, sessionID: session.ID, err: dbutil.ErrNotFound},
		{desc: "add candidate with invalid token", token: wrongValue, sessionID: session.ID, err: errors.ErrAuthentication},
	}

	for _, tc := range cases {
		err := svc.AddViewerCandidate(context.Background(), tc.token, tc.sessionID, candidate)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func webrtcMessage(thingID, kind, payload string) protomfx.Message {
	return protomfx.Message{
		ThingId:  thingID,
		Subtopic: "webrtc." + kind,
		Records:  []*protomfx.Record{{Payload: []byte(payload)}},
	}
}

func sessionOf(t *testing.T, pub *wmocks.Publisher) string {
	for _, c := range pub.Commands() {
		if c.Command.Subtopic == "webrtc.answer" {
			var body struct {
				SessionID string `json:"session_id"`
			}
			require.Nil(t, json.Unmarshal(c.Command.Payload, &body))
			return body.SessionID
		}
	}
	t.Fatal("no answer command recorded")
	return ""
}

func TestConsumeMessage(t *testing.T) {
	offerPayload := fmt.Sprintf(`{"session_id":"","sdp":%q}`, offer)
	candPayload := func(sid string) string {
		return fmt.Sprintf(`{"session_id":%q,"candidate":{"candidate":%q}}`, sid, candidate.Candidate)
	}

	t.Run("consume offer", func(t *testing.T) {
		svc, _, pub := newService(true)
		err := svc.ConsumeMessage("", webrtcMessage(thingID, "offer", offerPayload))
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

		cmds := pub.Commands()
		require.Len(t, cmds, 1)
		assert.Equal(t, nats.GetThingCommandsSubject(thingID, "webrtc.answer"), cmds[0].Subject)
		assert.Equal(t, thingID, cmds[0].Command.RecipientId)
		assert.Equal(t, "webrtc.answer", cmds[0].Command.Subtopic)
		assert.Equal(t, "webrtc", cmds[0].Command.Protocol)

		var body struct {
			SessionID  string             `json:"session_id"`
			SDP        string             `json:"sdp"`
			ICEServers []webrtc.ICEServer `json:"ice_servers"`
		}
		require.Nil(t, json.Unmarshal(cmds[0].Command.Payload, &body))
		assert.NotEmpty(t, body.SessionID)
		assert.Equal(t, "answer:"+body.SessionID, body.SDP)
		require.Len(t, body.ICEServers, 2)
		assert.True(t, strings.HasSuffix(body.ICEServers[1].Username, ":"+thingID), "turn username must carry the thing id")
	})

	cases := []struct {
		desc     string
		subtopic string
		payload  string
	}{
		{desc: "consume offer with malformed payload", subtopic: "webrtc.offer", payload: "{"},
		{desc: "consume offer with empty sdp", subtopic: "webrtc.offer", payload: `{"session_id":"","sdp":""}`},
		{desc: "consume offer with invalid sdp", subtopic: "webrtc.offer", payload: `{"sdp":"invalid"}`},
		{desc: "consume unknown webrtc kind", subtopic: "webrtc.foo", payload: "{}"},
		{desc: "consume bare webrtc subtopic", subtopic: "webrtc", payload: "{}"},
	}
	for _, tc := range cases {
		svc, _, pub := newService(false)
		msg := webrtcMessage(thingID, "", tc.payload)
		msg.Subtopic = tc.subtopic
		err := svc.ConsumeMessage("", msg)
		assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, errors.ErrMalformedEntity, err))
		assert.Empty(t, pub.Commands(), fmt.Sprintf("%s: nothing must be published\n", tc.desc))
	}

	t.Run("consume offer trickles server candidates", func(t *testing.T) {
		svc, sfu, pub := newService(false)
		require.Nil(t, svc.ConsumeMessage("", webrtcMessage(thingID, "offer", offerPayload)))
		sid := sessionOf(t, pub)

		sfu.EmitCandidate(sid, candidate)

		cmds := pub.Commands()
		require.Len(t, cmds, 2)
		assert.Equal(t, nats.GetThingCommandsSubject(thingID, "webrtc.candidate"), cmds[1].Subject)
		assert.Equal(t, thingID, cmds[1].Command.RecipientId)
		assert.Equal(t, "webrtc.candidate", cmds[1].Command.Subtopic)
		var got struct {
			SessionID string              `json:"session_id"`
			Candidate webrtc.ICECandidate `json:"candidate"`
		}
		require.Nil(t, json.Unmarshal(cmds[1].Command.Payload, &got))
		assert.Equal(t, sid, got.SessionID)
		assert.Equal(t, candidate.Candidate, got.Candidate.Candidate)
	})

	t.Run("consume offer publishes candidates gathered during publish after the answer", func(t *testing.T) {
		svc, sfu, pub := newService(false)
		sfu.EmitOnPublish(candidate)
		require.Nil(t, svc.ConsumeMessage("", webrtcMessage(thingID, "offer", offerPayload)))

		require.Eventually(t, func() bool { return len(pub.Commands()) == 2 }, time.Second, 10*time.Millisecond)
		cmds := pub.Commands()
		assert.Equal(t, "webrtc.answer", cmds[0].Command.Subtopic)
		assert.Equal(t, "webrtc.candidate", cmds[1].Command.Subtopic)
	})

	t.Run("consume invalid offer drops candidates gathered during publish", func(t *testing.T) {
		svc, sfu, pub := newService(false)
		sfu.EmitOnPublish(candidate)
		err := svc.ConsumeMessage("", webrtcMessage(thingID, "offer", fmt.Sprintf(`{"sdp":%q}`, invalidSDP)))
		assert.True(t, errors.Contains(err, errors.ErrMalformedEntity), fmt.Sprintf("expected %s got %s\n", errors.ErrMalformedEntity, err))

		assert.Never(t, func() bool { return len(pub.Commands()) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
	})

	candCases := []struct {
		desc    string
		thingID string
		unknown bool
		err     error
	}{
		{desc: "consume candidate for own session", thingID: thingID},
		{desc: "consume candidate for unknown session", thingID: thingID, unknown: true, err: dbutil.ErrNotFound},
		{desc: "consume candidate for session of another thing", thingID: otherThingID, err: dbutil.ErrNotFound},
	}
	for _, tc := range candCases {
		svc, _, pub := newService(false)
		require.Nil(t, svc.ConsumeMessage("", webrtcMessage(thingID, "offer", offerPayload)))
		sid := sessionOf(t, pub)
		if tc.unknown {
			sid = wrongValue
		}

		err := svc.ConsumeMessage("", webrtcMessage(tc.thingID, "candidate", candPayload(sid)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

	t.Run("consume config", func(t *testing.T) {
		svc, _, pub := newService(true)
		require.Nil(t, svc.ConsumeMessage("", webrtcMessage(thingID, "config", "{}")))

		cmds := pub.Commands()
		require.Len(t, cmds, 1)
		assert.Equal(t, nats.GetThingCommandsSubject(thingID, "webrtc.config"), cmds[0].Subject)
		assert.Equal(t, thingID, cmds[0].Command.RecipientId)
		assert.Equal(t, "webrtc.config", cmds[0].Command.Subtopic)
		var body struct {
			ICEServers []webrtc.ICEServer `json:"ice_servers"`
		}
		require.Nil(t, json.Unmarshal(cmds[0].Command.Payload, &body))
		require.Len(t, body.ICEServers, 2)
		assert.True(t, strings.HasSuffix(body.ICEServers[1].Username, ":"+thingID))
	})
}
