// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"sync"
	"time"
)

// TokenPair holds a short-lived access token and the long-lived refresh token
// used to mint replacements for it.
type TokenPair struct {
	AccessToken  string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// session caches the token pair from the most recent CreateToken so that
// sendRequest can renew an expired access token without the caller noticing.
//
// It models a single logged-in user: an SDK instance with AutoRefresh enabled
// tracks one session at a time, and a later CreateToken replaces the earlier
// one. Callers driving several users concurrently should use one SDK per user.
type session struct {
	mu     sync.RWMutex
	tokens TokenPair
}

func (s *session) set(tokens TokenPair) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = tokens
}

func (s *session) get() TokenPair {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens
}

// renewed swaps in a new pair only if the cached access token still matches
// stale, i.e. no other goroutine refreshed in the meantime. It reports the
// access token callers should retry with, and whether stale was still current.
func (s *session) renewed(stale string, tokens TokenPair) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tokens.AccessToken != stale {
		// Someone else already refreshed; reuse their token instead.
		return s.tokens.AccessToken, false
	}

	s.tokens = tokens
	return tokens.AccessToken, true
}
