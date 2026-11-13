// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package pion implements the webrtc.SFU port on top of Pion.
package pion

import (
	"net"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webrtc"
	"github.com/pion/ice/v4"
	"github.com/pion/logging"
	"github.com/pion/rtcp"
	pwebrtc "github.com/pion/webrtc/v4"
)

const rtpBufferSize = 1500

var _ webrtc.SFU = (*sfu)(nil)

// Config contains the Pion SFU settings.
type Config struct {
	// ListenAddr is the single UDP address all ICE traffic is multiplexed over.
	ListenAddr string
	// ICEServers are the STUN URLs used by the server itself.
	ICEServers []string
	// PublicIPs are advertised as NAT 1:1 host candidates when set.
	PublicIPs []string
	// IncludeLoopback and DisableMDNS exist for in-process tests.
	IncludeLoopback bool
	DisableMDNS     bool
}

type track struct {
	local *pwebrtc.TrackLocalStaticRTP
	ssrc  uint32
}

type session struct {
	id      string
	owner   string
	thingID string
	pc      *pwebrtc.PeerConnection
}

type publisher struct {
	session
	tracks  []track
	viewers map[string]*session
}

type sfu struct {
	api        *pwebrtc.API
	mux        ice.UDPMux
	iceServers []pwebrtc.ICEServer

	mu         sync.Mutex
	publishers map[string]*publisher
	sessions   map[string]*session
}

// New returns a Pion based SFU.
func New(cfg Config) (webrtc.SFU, error) {
	conn, err := net.ListenPacket("udp", cfg.ListenAddr)
	if err != nil {
		return nil, err
	}

	se := pwebrtc.SettingEngine{}
	mux := pwebrtc.NewICEUDPMux(logging.NewDefaultLoggerFactory().NewLogger("ice-mux"), conn)
	se.SetICEUDPMux(mux)
	se.SetIncludeLoopbackCandidate(cfg.IncludeLoopback)
	if cfg.DisableMDNS {
		se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	}
	if len(cfg.PublicIPs) > 0 {
		rule := pwebrtc.ICEAddressRewriteRule{
			External:        cfg.PublicIPs,
			AsCandidateType: pwebrtc.ICECandidateTypeHost,
			Mode:            pwebrtc.ICEAddressRewriteReplace,
		}
		if err := se.SetICEAddressRewriteRules(rule); err != nil {
			_ = mux.Close()
			return nil, err
		}
	}

	s := &sfu{
		api:        pwebrtc.NewAPI(pwebrtc.WithSettingEngine(se)),
		mux:        mux,
		publishers: make(map[string]*publisher),
		sessions:   make(map[string]*session),
	}
	if len(cfg.ICEServers) > 0 {
		s.iceServers = []pwebrtc.ICEServer{{URLs: cfg.ICEServers}}
	}

	return s, nil
}

func (s *sfu) newPeerConnection() (*pwebrtc.PeerConnection, error) {
	return s.api.NewPeerConnection(pwebrtc.Configuration{ICEServers: s.iceServers})
}

func (s *sfu) Publish(sessionID, thingID, offer string, onCandidate func(webrtc.ICECandidate)) (string, error) {
	pc, err := s.newPeerConnection()
	if err != nil {
		return "", err
	}

	pub := &publisher{
		session: session{id: sessionID, owner: thingID, thingID: thingID, pc: pc},
		viewers: make(map[string]*session),
	}

	pc.OnICECandidate(func(c *pwebrtc.ICECandidate) {
		if c == nil || onCandidate == nil {
			return
		}
		init := c.ToJSON()
		onCandidate(webrtc.ICECandidate{
			Candidate:        init.Candidate,
			SDPMid:           init.SDPMid,
			SDPMLineIndex:    init.SDPMLineIndex,
			UsernameFragment: init.UsernameFragment,
		})
	})

	pc.OnTrack(func(remote *pwebrtc.TrackRemote, _ *pwebrtc.RTPReceiver) {
		local, err := pwebrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, remote.ID(), remote.StreamID())
		if err != nil {
			return
		}

		s.mu.Lock()
		if s.publishers[thingID] != pub {
			s.mu.Unlock()
			return
		}
		pub.tracks = append(pub.tracks, track{local: local, ssrc: uint32(remote.SSRC())})
		s.mu.Unlock()

		buf := make([]byte, rtpBufferSize)
		for {
			n, _, err := remote.Read(buf)
			if err != nil {
				return
			}
			if _, err := local.Write(buf[:n]); err != nil {
				return
			}
		}
	})

	pc.OnConnectionStateChange(func(state pwebrtc.PeerConnectionState) {
		if state == pwebrtc.PeerConnectionStateFailed || state == pwebrtc.PeerConnectionStateClosed {
			s.removePublisher(pub)
		}
	})

	answer, err := answerOffer(pc, offer)
	if err != nil {
		_ = pc.Close()
		return "", err
	}

	s.mu.Lock()
	prev := s.publishers[thingID]
	s.publishers[thingID] = pub
	s.sessions[sessionID] = &pub.session
	s.mu.Unlock()

	if prev != nil {
		s.removePublisher(prev)
	}

	return answer, nil
}

func (s *sfu) View(sessionID, thingID, userID, offer string) (string, error) {
	s.mu.Lock()
	pub := s.publishers[thingID]
	var tracks []track
	if pub != nil {
		tracks = append(tracks, pub.tracks...)
	}
	s.mu.Unlock()

	if len(tracks) == 0 {
		return "", dbutil.ErrNotFound
	}

	pc, err := s.newPeerConnection()
	if err != nil {
		return "", err
	}

	sess := &session{id: sessionID, owner: userID, thingID: thingID, pc: pc}

	for _, t := range tracks {
		sender, err := pc.AddTrack(t.local)
		if err != nil {
			_ = pc.Close()
			return "", err
		}
		go forwardPLI(sender, pub, t.ssrc)
	}

	pc.OnConnectionStateChange(func(state pwebrtc.PeerConnectionState) {
		if state == pwebrtc.PeerConnectionStateFailed || state == pwebrtc.PeerConnectionStateClosed {
			s.removeViewer(pub, sess)
		}
	})

	gathered := pwebrtc.GatheringCompletePromise(pc)
	if _, err := answerOffer(pc, offer); err != nil {
		_ = pc.Close()
		return "", err
	}
	<-gathered

	s.mu.Lock()
	if s.publishers[thingID] != pub {
		s.mu.Unlock()
		_ = pc.Close()
		return "", dbutil.ErrNotFound
	}
	pub.viewers[sessionID] = sess
	s.sessions[sessionID] = sess
	s.mu.Unlock()

	for _, t := range tracks {
		requestKeyFrame(pub.pc, t.ssrc)
	}

	return pc.LocalDescription().SDP, nil
}

func (s *sfu) AddCandidate(sessionID, ownerID string, c webrtc.ICECandidate) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if !ok || sess.owner != ownerID {
		return dbutil.ErrNotFound
	}

	if err := sess.pc.AddICECandidate(pwebrtc.ICECandidateInit{
		Candidate:        c.Candidate,
		SDPMid:           c.SDPMid,
		SDPMLineIndex:    c.SDPMLineIndex,
		UsernameFragment: c.UsernameFragment,
	}); err != nil {
		return errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return nil
}

func (s *sfu) Close() error {
	s.mu.Lock()
	var pcs []*pwebrtc.PeerConnection
	for _, sess := range s.sessions {
		pcs = append(pcs, sess.pc)
	}
	s.publishers = make(map[string]*publisher)
	s.sessions = make(map[string]*session)
	s.mu.Unlock()

	for _, pc := range pcs {
		_ = pc.Close()
	}

	return s.mux.Close()
}

func (s *sfu) removePublisher(pub *publisher) {
	s.mu.Lock()
	if s.publishers[pub.thingID] == pub {
		delete(s.publishers, pub.thingID)
	}
	delete(s.sessions, pub.id)
	viewers := pub.viewers
	pub.viewers = make(map[string]*session)
	for id := range viewers {
		delete(s.sessions, id)
	}
	s.mu.Unlock()

	_ = pub.pc.Close()
	for _, v := range viewers {
		_ = v.pc.Close()
	}
}

func (s *sfu) removeViewer(pub *publisher, v *session) {
	s.mu.Lock()
	delete(pub.viewers, v.id)
	delete(s.sessions, v.id)
	s.mu.Unlock()

	_ = v.pc.Close()
}

func answerOffer(pc *pwebrtc.PeerConnection, offer string) (string, error) {
	if err := pc.SetRemoteDescription(pwebrtc.SessionDescription{Type: pwebrtc.SDPTypeOffer, SDP: offer}); err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return "", errors.Wrap(errors.ErrMalformedEntity, err)
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		return "", err
	}

	return pc.LocalDescription().SDP, nil
}

func forwardPLI(sender *pwebrtc.RTPSender, pub *publisher, ssrc uint32) {
	for {
		pkts, _, err := sender.ReadRTCP()
		if err != nil {
			return
		}
		for _, p := range pkts {
			if _, ok := p.(*rtcp.PictureLossIndication); ok {
				requestKeyFrame(pub.pc, ssrc)
			}
		}
	}
}

func requestKeyFrame(pc *pwebrtc.PeerConnection, ssrc uint32) {
	_ = pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}})
}
