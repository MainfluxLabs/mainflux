// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	piondtls "github.com/pion/dtls/v3"
	udpclient "github.com/plgd-dev/go-coap/v3/udp/client"
)

// cidSize is the length of the connection ID the server asks clients to send.
const cidSize = 8

type thingKeyContextKey struct{}

// DTLSOptions returns the DTLS server options: PSK authentication, the cipher suites
// constrained devices support, and connection IDs, which keep a session alive when
// NAT changes the client's address (RFC 9146).
func DTLSOptions(things domain.ThingsClient, logger log.Logger) []piondtls.ServerOption {
	return []piondtls.ServerOption{
		piondtls.WithPSK(PSK(things, logger)),
		piondtls.WithCipherSuites(piondtls.TLS_PSK_WITH_AES_128_CCM_8, piondtls.TLS_PSK_WITH_AES_128_GCM_SHA256),
		piondtls.WithConnectionIDGenerator(piondtls.RandomCIDGenerator(cidSize)),
	}
}

// pskSize is the PSK length most constrained DTLS stacks accept at most.
const pskSize = 32

// PSK returns the DTLS PSK callback, which takes the thing ID as the PSK identity
// and DerivePSK of the thing key as the pre-shared key.
func PSK(things domain.ThingsClient, logger log.Logger) piondtls.PSKCallback {
	return func(identity []byte) ([]byte, error) {
		key, err := things.GetKeyByThingID(context.Background(), string(identity))
		if err != nil {
			logger.Warn(fmt.Sprintf("DTLS handshake rejected for thing %s: %s", identity, err))
			return nil, err
		}

		return DerivePSK(key.Value), nil
	}
}

// DerivePSK returns the pre-shared key of a thing: the first 32 hex characters of the
// SHA-256 hash of its key. The key itself is longer than constrained DTLS stacks accept,
// and hex keeps the PSK usable as a string in clients and device configs.
func DerivePSK(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return []byte(hex.EncodeToString(hash[:])[:pskSize])
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
