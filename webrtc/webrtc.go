// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webrtc

import "time"

// ICEServer is an ICE server entry in the W3C RTCIceServer shape.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// ICECandidate is an ICE candidate in the W3C RTCIceCandidateInit shape.
type ICECandidate struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment *string `json:"usernameFragment,omitempty"`
}

// Session represents a viewer session along with the SDP answer.
type Session struct {
	ID  string `json:"session_id"`
	SDP string `json:"sdp"`
}

// Config contains the ICE server configuration handed out to clients.
type Config struct {
	STUNURLs   []string
	TURNURLs   []string
	TURNSecret string
	TURNTTL    time.Duration
}

// SFU specifies the selective forwarding unit used by the service.
// Session owners are the thing ID for publishers and the user ID for viewers.
type SFU interface {
	// Publish creates a publisher session for the thing, replacing the previous one,
	// and returns the SDP answer. The callback receives server ICE candidates.
	Publish(sessionID, thingID, offer string, onCandidate func(ICECandidate)) (string, error)

	// View attaches a viewer session to the active publisher of the thing
	// and returns the SDP answer.
	View(sessionID, thingID, userID, offer string) (string, error)

	// AddCandidate adds a remote ICE candidate to the session owned by ownerID.
	AddCandidate(sessionID, ownerID string, candidate ICECandidate) error

	// Close closes all sessions.
	Close() error
}
