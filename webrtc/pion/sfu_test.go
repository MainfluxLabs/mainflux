// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package pion_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webrtc"
	"github.com/MainfluxLabs/mainflux/webrtc/pion"
	"github.com/pion/ice/v4"
	"github.com/pion/rtp"
	pwebrtc "github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID    = "thing-1"
	otherThing = "thing-2"
	userID     = "user-1"
	otherUser  = "user-2"
	waitFor    = 10 * time.Second
	invalidSDP = "not an sdp"
)

func newSFU(t *testing.T) webrtc.SFU {
	sfu, err := pion.New(pion.Config{
		ListenAddr:      "127.0.0.1:0",
		IncludeLoopback: true,
		DisableMDNS:     true,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error creating sfu: %s", err))
	t.Cleanup(func() { _ = sfu.Close() })

	return sfu
}

func newClientPC(t *testing.T) *pwebrtc.PeerConnection {
	se := pwebrtc.SettingEngine{}
	se.SetIncludeLoopbackCandidate(true)
	se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)

	pc, err := pwebrtc.NewAPI(pwebrtc.WithSettingEngine(se)).NewPeerConnection(pwebrtc.Configuration{})
	require.Nil(t, err, fmt.Sprintf("unexpected error creating client: %s", err))
	t.Cleanup(func() { _ = pc.Close() })

	return pc
}

func gatheredOffer(t *testing.T, pc *pwebrtc.PeerConnection) string {
	offer, err := pc.CreateOffer(nil)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating offer: %s", err))

	done := pwebrtc.GatheringCompletePromise(pc)
	require.Nil(t, pc.SetLocalDescription(offer))
	<-done

	return pc.LocalDescription().SDP
}

func toICE(c webrtc.ICECandidate) pwebrtc.ICECandidateInit {
	return pwebrtc.ICECandidateInit{
		Candidate:        c.Candidate,
		SDPMid:           c.SDPMid,
		SDPMLineIndex:    c.SDPMLineIndex,
		UsernameFragment: c.UsernameFragment,
	}
}

type publisher struct {
	pc    *pwebrtc.PeerConnection
	offer string
	stop  chan struct{}

	mu      sync.Mutex
	ready   bool
	pending []webrtc.ICECandidate
}

func newPublisher(t *testing.T) *publisher {
	pc := newClientPC(t)
	track, err := pwebrtc.NewTrackLocalStaticRTP(
		pwebrtc.RTPCodecCapability{MimeType: pwebrtc.MimeTypeVP8}, "video", "stream")
	require.Nil(t, err)
	_, err = pc.AddTrack(track)
	require.Nil(t, err)

	p := &publisher{pc: pc, stop: make(chan struct{})}
	p.offer = gatheredOffer(t, pc)

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		var seq uint16
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				seq++
				_ = track.WriteRTP(&rtp.Packet{
					Header:  rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: seq, Timestamp: uint32(seq) * 3000},
					Payload: []byte{0x10, 0x00, 0x00, 0x00},
				})
			}
		}
	}()
	t.Cleanup(func() { close(p.stop) })

	return p
}

func (p *publisher) onCandidate(c webrtc.ICECandidate) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.ready {
		p.pending = append(p.pending, c)
		return
	}
	_ = p.pc.AddICECandidate(toICE(c))
}

func (p *publisher) applyAnswer(t *testing.T, sdp string) {
	require.Nil(t, p.pc.SetRemoteDescription(pwebrtc.SessionDescription{Type: pwebrtc.SDPTypeAnswer, SDP: sdp}))

	p.mu.Lock()
	defer p.mu.Unlock()
	p.ready = true
	for _, c := range p.pending {
		_ = p.pc.AddICECandidate(toICE(c))
	}
	p.pending = nil
}

func publish(t *testing.T, sfu webrtc.SFU, sessionID, thing string) *publisher {
	p := newPublisher(t)
	answer, err := sfu.Publish(sessionID, thing, p.offer, p.onCandidate)
	require.Nil(t, err, fmt.Sprintf("unexpected error publishing: %s", err))
	p.applyAnswer(t, answer)

	return p
}

type viewer struct {
	pc     *pwebrtc.PeerConnection
	offer  string
	tracks chan struct{}
}

func newViewer(t *testing.T) *viewer {
	pc := newClientPC(t)
	_, err := pc.AddTransceiverFromKind(pwebrtc.RTPCodecTypeVideo,
		pwebrtc.RTPTransceiverInit{Direction: pwebrtc.RTPTransceiverDirectionRecvonly})
	require.Nil(t, err)

	v := &viewer{pc: pc, tracks: make(chan struct{}, 1)}
	pc.OnTrack(func(*pwebrtc.TrackRemote, *pwebrtc.RTPReceiver) {
		select {
		case v.tracks <- struct{}{}:
		default:
		}
	})
	v.offer = gatheredOffer(t, pc)

	return v
}

func (v *viewer) viewWhenReady(sfu webrtc.SFU, sessionID, thing, user string) (string, error) {
	deadline := time.Now().Add(waitFor)
	for {
		answer, err := sfu.View(sessionID, thing, user, v.offer)
		if !errors.Contains(err, dbutil.ErrNotFound) || time.Now().After(deadline) {
			return answer, err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestPublish(t *testing.T) {
	sfu := newSFU(t)
	validOffer := newPublisher(t).offer

	cases := []struct {
		desc  string
		offer string
		err   error
	}{
		{desc: "publish valid offer", offer: validOffer, err: nil},
		{desc: "publish invalid sdp", offer: invalidSDP, err: errors.ErrMalformedEntity},
	}

	for _, tc := range cases {
		answer, err := sfu.Publish("session-"+tc.desc, thingID, tc.offer, func(webrtc.ICECandidate) {})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.NotEmpty(t, answer, fmt.Sprintf("%s: expected an sdp answer\n", tc.desc))
		}
	}

	prev := "session-prev"
	publish(t, sfu, prev, otherThing)
	err := sfu.AddCandidate(prev, otherThing, webrtc.ICECandidate{Candidate: "candidate:0 1 udp 1 127.0.0.1 9 typ host"})
	assert.Nil(t, err, fmt.Sprintf("publish: expected live session, got %s\n", err))

	publish(t, sfu, "session-next", otherThing)
	err = sfu.AddCandidate(prev, otherThing, webrtc.ICECandidate{Candidate: "candidate:0 1 udp 1 127.0.0.1 9 typ host"})
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound),
		fmt.Sprintf("publish replaces previous session of thing: expected %s got %s\n", dbutil.ErrNotFound, err))
}

func TestView(t *testing.T) {
	sfu := newSFU(t)
	publish(t, sfu, "pub-1", thingID)

	cases := []struct {
		desc   string
		thing  string
		offer  string
		track  bool
		err    error
		viewer *viewer
	}{
		{desc: "view thing without publisher", thing: otherThing, viewer: newViewer(t), err: dbutil.ErrNotFound},
		{desc: "view thing with publisher receives track", thing: thingID, track: true, viewer: newViewer(t), err: nil},
		{desc: "view with invalid sdp", thing: thingID, offer: invalidSDP, err: errors.ErrMalformedEntity},
	}

	for _, tc := range cases {
		var (
			answer string
			err    error
		)
		switch {
		case tc.viewer != nil && tc.thing == otherThing:
			answer, err = sfu.View("view-"+tc.desc, tc.thing, userID, tc.viewer.offer)
		case tc.viewer != nil:
			answer, err = tc.viewer.viewWhenReady(sfu, "view-"+tc.desc, tc.thing, userID)
		default:
			deadline := time.Now().Add(waitFor)
			for {
				answer, err = sfu.View("view-"+tc.desc, tc.thing, userID, tc.offer)
				if !errors.Contains(err, dbutil.ErrNotFound) || time.Now().After(deadline) {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		if tc.track {
			require.Nil(t, tc.viewer.pc.SetRemoteDescription(
				pwebrtc.SessionDescription{Type: pwebrtc.SDPTypeAnswer, SDP: answer}))
			select {
			case <-tc.viewer.tracks:
			case <-time.After(waitFor):
				t.Errorf("%s: viewer did not receive a track\n", tc.desc)
			}
		}
	}
}

func TestAddCandidate(t *testing.T) {
	sfu := newSFU(t)
	publish(t, sfu, "pub-1", thingID)
	candidate := webrtc.ICECandidate{Candidate: "candidate:0 1 udp 1 127.0.0.1 9 typ host"}

	cases := []struct {
		desc      string
		sessionID string
		owner     string
		err       error
	}{
		{desc: "add candidate to unknown session", sessionID: "unknown", owner: thingID, err: dbutil.ErrNotFound},
		{desc: "add candidate with wrong owner", sessionID: "pub-1", owner: otherThing, err: dbutil.ErrNotFound},
		{desc: "add candidate to own session", sessionID: "pub-1", owner: thingID, err: nil},
	}

	for _, tc := range cases {
		err := sfu.AddCandidate(tc.sessionID, tc.owner, candidate)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestPublisherClose(t *testing.T) {
	sfu := newSFU(t)
	pub := publish(t, sfu, "pub-1", thingID)

	v := newViewer(t)
	answer, err := v.viewWhenReady(sfu, "view-1", thingID, otherUser)
	require.Nil(t, err, fmt.Sprintf("unexpected error viewing: %s", err))
	require.Nil(t, v.pc.SetRemoteDescription(pwebrtc.SessionDescription{Type: pwebrtc.SDPTypeAnswer, SDP: answer}))

	require.Nil(t, pub.pc.Close())

	candidate := webrtc.ICECandidate{Candidate: "candidate:0 1 udp 1 127.0.0.1 9 typ host"}
	deadline := time.Now().Add(waitFor)
	for {
		err = sfu.AddCandidate("view-1", otherUser, candidate)
		if errors.Contains(err, dbutil.ErrNotFound) || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound),
		fmt.Sprintf("closing publisher removes its viewers: expected %s got %v\n", dbutil.ErrNotFound, err))

	_, err = sfu.View("view-2", thingID, userID, v.offer)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound),
		fmt.Sprintf("view after publisher close: expected %s got %v\n", dbutil.ErrNotFound, err))
}

type dataClient struct {
	pc    *pwebrtc.PeerConnection
	dc    *pwebrtc.DataChannel
	offer string
	open  chan struct{}
	msgs  chan pwebrtc.DataChannelMessage
}

func newDataClient(t *testing.T, label string) *dataClient {
	pc := newClientPC(t)
	dc, err := pc.CreateDataChannel(label, nil)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating data channel: %s", err))

	c := &dataClient{pc: pc, dc: dc, open: make(chan struct{}), msgs: make(chan pwebrtc.DataChannelMessage, 10)}
	dc.OnOpen(func() { close(c.open) })
	dc.OnMessage(func(m pwebrtc.DataChannelMessage) { c.msgs <- m })
	c.offer = gatheredOffer(t, pc)

	return c
}

func (c *dataClient) waitOpen(t *testing.T, desc string) {
	select {
	case <-c.open:
	case <-time.After(waitFor):
		t.Fatalf("%s: data channel did not open\n", desc)
	}
}

func TestDataChannel(t *testing.T) {
	sfu := newSFU(t)
	const label = "commands"

	dev := newDataClient(t, label)
	p := &publisher{pc: dev.pc}
	answer, err := sfu.Publish("pub-1", thingID, dev.offer, p.onCandidate)
	require.Nil(t, err, fmt.Sprintf("unexpected error publishing: %s", err))
	p.applyAnswer(t, answer)
	dev.waitOpen(t, "publish data-only thing")

	ui := newDataClient(t, label)
	deadline := time.Now().Add(waitFor)
	for {
		answer, err = sfu.View("view-1", thingID, userID, ui.offer)
		if !errors.Contains(err, dbutil.ErrNotFound) || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Nil(t, err, fmt.Sprintf("view data-only thing: expected no error got %s", err))
	require.Nil(t, ui.pc.SetRemoteDescription(pwebrtc.SessionDescription{Type: pwebrtc.SDPTypeAnswer, SDP: answer}))
	ui.waitOpen(t, "view data-only thing")

	cases := []struct {
		desc     string
		from     *dataClient
		to       *dataClient
		data     []byte
		isString bool
	}{
		{desc: "relay text from viewer to thing", from: ui, to: dev, data: []byte(`{"cmd":"reboot"}`), isString: true},
		{desc: "relay binary from viewer to thing", from: ui, to: dev, data: []byte{0x01, 0x02}, isString: false},
		{desc: "relay text from thing to viewer", from: dev, to: ui, data: []byte(`{"ok":true}`), isString: true},
		{desc: "relay binary from thing to viewer", from: dev, to: ui, data: []byte{0x03, 0x04}, isString: false},
	}

	for _, tc := range cases {
		if tc.isString {
			err = tc.from.dc.SendText(string(tc.data))
		} else {
			err = tc.from.dc.Send(tc.data)
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error sending: %s", tc.desc, err))

		select {
		case m := <-tc.to.msgs:
			assert.Equal(t, tc.data, m.Data, fmt.Sprintf("%s: expected %q got %q\n", tc.desc, tc.data, m.Data))
			assert.Equal(t, tc.isString, m.IsString, fmt.Sprintf("%s: expected string %t got %t\n", tc.desc, tc.isString, m.IsString))
		case <-time.After(waitFor):
			t.Errorf("%s: message was not relayed\n", tc.desc)
		}
	}
}
