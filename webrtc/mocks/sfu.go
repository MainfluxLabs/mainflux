// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/webrtc"
)

const (
	invalidOffer = "invalid"
	emitWait     = 50 * time.Millisecond
)

var _ webrtc.SFU = (*SFU)(nil)

type session struct {
	thingID     string
	owner       string
	onCandidate func(webrtc.ICECandidate)
}

// SFU is an in-memory fake of webrtc.SFU.
type SFU struct {
	mu            sync.Mutex
	sessions      map[string]session
	publishers    map[string]string
	emitOnPublish *webrtc.ICECandidate
}

// NewSFU returns an in-memory fake of the SFU.
func NewSFU() *SFU {
	return &SFU{
		sessions:   make(map[string]session),
		publishers: make(map[string]string),
	}
}

func (s *SFU) Publish(sessionID, thingID, offer string, onCandidate func(webrtc.ICECandidate)) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.emitOnPublish != nil {
		c := *s.emitOnPublish
		done := make(chan struct{})
		go func() {
			onCandidate(c)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(emitWait):
		}
	}

	if offer == invalidOffer {
		return "", errors.ErrMalformedEntity
	}

	if prev, ok := s.publishers[thingID]; ok {
		delete(s.sessions, prev)
	}

	s.publishers[thingID] = sessionID
	s.sessions[sessionID] = session{thingID: thingID, owner: thingID, onCandidate: onCandidate}

	return "answer:" + sessionID, nil
}

func (s *SFU) View(sessionID, thingID, userID, offer string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.publishers[thingID]; !ok {
		return "", dbutil.ErrNotFound
	}

	if offer == invalidOffer {
		return "", errors.ErrMalformedEntity
	}

	s.sessions[sessionID] = session{thingID: thingID, owner: userID}

	return "answer:" + sessionID, nil
}

func (s *SFU) AddCandidate(sessionID, ownerID string, _ webrtc.ICECandidate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok || sess.owner != ownerID {
		return dbutil.ErrNotFound
	}

	return nil
}

// EmitCandidate simulates a server ICE candidate gathered for the publisher session.
func (s *SFU) EmitCandidate(sessionID string, c webrtc.ICECandidate) {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if ok && sess.onCandidate != nil {
		sess.onCandidate(c)
	}
}

// EmitOnPublish makes Publish gather the candidate while the offer is being answered,
// as Pion does, giving the callback a bounded time to return before Publish returns.
func (s *SFU) EmitOnPublish(c webrtc.ICECandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.emitOnPublish = &c
}

func (s *SFU) Close() error {
	return nil
}
