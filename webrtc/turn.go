// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package webrtc

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"time"
)

func iceServers(cfg Config, id string, now time.Time) []ICEServer {
	servers := []ICEServer{{URLs: cfg.STUNURLs}}
	if len(cfg.TURNURLs) == 0 || cfg.TURNSecret == "" {
		return servers
	}

	username := strconv.FormatInt(now.Add(cfg.TURNTTL).Unix(), 10) + ":" + id
	mac := hmac.New(sha1.New, []byte(cfg.TURNSecret))
	mac.Write([]byte(username))

	return append(servers, ICEServer{
		URLs:       cfg.TURNURLs,
		Username:   username,
		Credential: base64.StdEncoding.EncodeToString(mac.Sum(nil)),
	})
}
