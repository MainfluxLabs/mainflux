// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	piondtls "github.com/pion/dtls/v3"
	udpclient "github.com/plgd-dev/go-coap/v3/udp/client"
)

type thingKeyContextKey struct{}

// PSK returns the DTLS PSK callback, which takes the thing ID as the PSK identity
// and the thing key as the pre-shared key.
func PSK(things domain.ThingsClient, logger log.Logger) piondtls.PSKCallback {
	return func(identity []byte) ([]byte, error) {
		key, err := things.GetKeyByThingID(context.Background(), string(identity))
		if err != nil {
			logger.Warn(fmt.Sprintf("DTLS handshake rejected for thing %s: %s", identity, err))
			return nil, err
		}

		return []byte(key.Value), nil
	}
}

// OnNewDTLSConn stores the key of the thing that completed the handshake, so that
// requests on the connection need no auth query.
func OnNewDTLSConn(things domain.ThingsClient, logger log.Logger) func(*udpclient.Conn) {
	return func(conn *udpclient.Conn) {
		dtlsConn, ok := conn.NetConn().(*piondtls.Conn)
		if !ok {
			return
		}
		state, ok := dtlsConn.ConnectionState()
		if !ok {
			_ = conn.Close()
			return
		}

		key, err := things.GetKeyByThingID(context.Background(), string(state.IdentityHint))
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to retrieve key of thing %s: %s", state.IdentityHint, err))
			_ = conn.Close()
			return
		}

		conn.SetContextValue(thingKeyContextKey{}, key)
	}
}

// sessionKey returns the thing key of the DTLS session the request came over.
func sessionKey(ctx context.Context) (domain.ThingKey, bool) {
	key, ok := ctx.Value(thingKeyContextKey{}).(domain.ThingKey)
	return key, ok
}
