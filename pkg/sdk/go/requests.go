// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

type groupThingsReq struct {
	Things []string `json:"things"`
}

type groupChannelsReq struct {
	Channels []string `json:"channels"`
}

// UserPasswordReq contains old and new passwords
type UserPasswordReq struct {
	OldPassword string `json:"old_password,omitempty"`
	Password    string `json:"password,omitempty"`
}

// ConnectionIDs contains ID lists of things and channel to be connected
type ConnectionIDs struct {
	ChannelID string   `json:"channel_id"`
	ThingIDs  []string `json:"thing_ids"`
}

// DisconnectionIDs contains ID lists of things and channels to be disconnected
type DisconnectionIDs struct {
	ChannelIDs []string `json:"channel_ids"`
	ThingIDs   []string `json:"thing_ids"`
}
