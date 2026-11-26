// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	certsevents "github.com/MainfluxLabs/mainflux/certs/events"
	"github.com/MainfluxLabs/mainflux/certs/mocks"
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	mfcron "github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email           = "user@example.com"
	token           = email
	thingKey        = "thing-key"
	thingID         = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	unscopedThingID = "a6e1c2f4-3b5d-4f7e-9c8a-1d2e3f4a5b6c"
	groupID         = "c5a1e8f2-6b3d-4a9e-8f7c-2d1b0e9a8c7f"
	orgID           = "f8d2c1b4-7a6e-4d3c-9b8a-5e4f3d2c1b0a"
	ttl             = "1h"
	longTTL         = "2000h"
	keyBits         = 2048
	keyType         = "rsa"
)

var user = users.User{
	ID:    "574106f7-030e-4881-8ab0-151195c29f94",
	Email: email,
}

type testEnv struct {
	svc    certs.Service
	auth   domain.AuthClient
	things domain.ThingsClient
	repo   certs.Repository
	pki    pki.Agent
}

// newEnv returns a certs service whose CRL is written to crlPath, or not written when it is empty.
func newEnv(t *testing.T, crlPath string) testEnv {
	t.Helper()
	ac := pkgmocks.NewAuthService("", []users.User{user}, nil)

	// The things mock authorizes by token and looks things up by ID.
	ths := map[string]domain.Thing{
		token:           {ID: thingID, Key: thingKey, GroupID: groupID},
		thingID:         {ID: thingID, Key: thingKey, GroupID: groupID},
		unscopedThingID: {ID: unscopedThingID, Key: thingKey},
	}
	groups := map[string]domain.Group{
		groupID: {ID: groupID, OrgID: orgID},
	}
	tc := pkgmocks.NewThingsServiceClient(map[string]domain.Profile{}, ths, groups)

	pkiAgent, err := pki.NewAgent(newCA(t))
	require.NoError(t, err)

	repo := mocks.NewCertsRepository()
	svc := certs.New(ac, tc, repo, certs.Config{CRLPath: crlPath}, pkiAgent)

	return testEnv{
		svc:    svc,
		auth:   ac,
		things: tc,
		repo:   repo,
		pki:    pkiAgent,
	}
}

func newPublisher(t *testing.T) events.Publisher {
	t.Helper()
	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.CertsStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	return pub
}

// withEventStore wraps the service in the event store middleware backed by the test Redis.
func (env testEnv) withEventStore(t *testing.T) certs.Service {
	t.Helper()
	return certsevents.NewEventStoreMiddleware(env.svc, env.things, newPublisher(t))
}

// identityCtx returns a context carrying the token owner's identity, as the HTTP API sets it.
func (env testEnv) identityCtx(token string) context.Context {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := authn.HTTPTokenToContext(context.Background(), req)

	var identityCtx context.Context
	capture := func(ctx context.Context, _ any) (any, error) {
		identityCtx = ctx
		return nil, nil
	}
	_, _ = authn.IdentityMiddleware(env.auth, logger.NewMock())(capture)(ctx, nil)

	return identityCtx
}

func newCA(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Mainflux Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
}

func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.CertsStream, lastID},
		Count:   1,
		Block:   time.Second,
	}).Val()

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, lastID
	}

	msg := streams[0].Messages[0]
	assert.NotEmpty(t, msg.Values["occurred_at"], desc)
	delete(msg.Values, "occurred_at")

	return msg.Values, msg.ID
}

// certEvent returns the expected fields of an event the user performed on a thing in the test group.
func certEvent(operation string, fields map[string]any) map[string]any {
	event := map[string]any{
		"operation":               operation,
		"jwt_identity_user_id":    user.ID,
		"jwt_identity_user_email": user.Email,
		"evt_org_id":              orgID,
		"evt_group_id":            groupID,
	}
	for k, v := range fields {
		event[k] = v
	}

	return event
}

func TestIssueCert(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t, "")
	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		token     string
		thingID   string
		err       error
		published bool
	}{
		{
			desc:      "issue cert with invalid token",
			token:     "invalid",
			thingID:   thingID,
			err:       errors.ErrAuthorization,
			published: false,
		},
		{
			desc:      "issue cert",
			token:     token,
			thingID:   thingID,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		cert, err := svc.IssueCert(env.identityCtx(tc.token), tc.token, tc.thingID, ttl, keyBits, keyType)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = certEvent(events.CertIssue, map[string]any{
				"serial":   cert.Serial,
				"thing_id": tc.thingID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRotateCert(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t, "")

	oldCert, err := env.svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		serial    string
		thingID   string
		err       error
		published bool
	}{
		{
			desc:      "rotate cert of another thing",
			serial:    oldCert.Serial,
			thingID:   unscopedThingID,
			err:       errors.ErrAuthorization,
			published: false,
		},
		{
			desc:      "rotate cert",
			serial:    oldCert.Serial,
			thingID:   thingID,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		cert, err := svc.RotateCert(env.identityCtx(token), token, tc.serial, tc.thingID, ttl, keyBits, keyType)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = certEvent(events.CertRotate, map[string]any{
				"serial":         cert.Serial,
				"revoked_serial": tc.serial,
				"thing_id":       tc.thingID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRevokeCert(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t, "")

	cert, err := env.svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		token     string
		serial    string
		err       error
		published bool
	}{
		{
			desc:      "revoke cert with invalid token",
			token:     "invalid",
			serial:    cert.Serial,
			err:       errors.ErrAuthorization,
			published: false,
		},
		{
			desc:      "revoke cert",
			token:     token,
			serial:    cert.Serial,
			err:       nil,
			published: true,
		},
		{
			desc:      "revoke revoked cert",
			token:     token,
			serial:    cert.Serial,
			err:       dbutil.ErrNotFound,
			published: false,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.RevokeCert(env.identityCtx(tc.token), tc.token, tc.serial)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = certEvent(events.CertRevoke, map[string]any{
				"serial":   tc.serial,
				"thing_id": thingID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRenewCert(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t, "")

	expiring, err := env.svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	valid, err := env.svc.IssueCert(context.Background(), token, thingID, longTTL, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	svc := env.withEventStore(t)

	cases := []struct {
		desc      string
		serial    string
		err       error
		published bool
	}{
		{
			desc:      "renew cert not eligible for renewal",
			serial:    valid.Serial,
			err:       certs.ErrNotEligibleForRenewal,
			published: false,
		},
		{
			desc:      "renew expiring cert",
			serial:    expiring.Serial,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		cert, err := svc.RenewCert(env.identityCtx(token), token, tc.serial)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = certEvent(events.CertRenew, map[string]any{
				"serial":         cert.Serial,
				"revoked_serial": tc.serial,
				"thing_id":       thingID,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestSchedulerRenewal(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t, "")

	cases := []struct {
		desc    string
		thingID string
		groupID string
		orgID   string
	}{
		{
			desc:    "renew expiring cert of a thing in an org",
			thingID: thingID,
			groupID: groupID,
			orgID:   orgID,
		},
		{
			desc:    "renew expiring cert of a thing whose scope can't be looked up",
			thingID: unscopedThingID,
			groupID: "",
			orgID:   "",
		},
	}

	lastID := "0"
	for _, tc := range cases {
		expiring := certs.Cert{
			ThingID:   tc.thingID,
			Serial:    tc.thingID + "-serial",
			ExpiresAt: time.Now().Add(time.Hour),
		}
		_, err := env.repo.Save(context.Background(), expiring)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		// Start renews expiring certificates right away, then waits for the schedule.
		ctx, cancel := context.WithCancel(context.Background())
		scheduler := certs.NewCertScheduler(env.repo, env.pki, env.things, pkgmocks.NewPublisher(), newPublisher(t), "", logger.NewMock())
		done := make(chan error)
		go func() {
			done <- scheduler.Start(ctx, mfcron.Scheduler{TimeZone: "UTC", Frequency: mfcron.DailyFreq, DayTime: "02:00"})
		}()

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		cancel()
		require.Nil(t, <-done, fmt.Sprintf("%s: unexpected scheduler error", tc.desc))

		renewed, err := env.repo.RetrieveByThing(context.Background(), tc.thingID, certs.PageMetadata{Limit: 10})
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		require.Len(t, renewed.Certs, 1, tc.desc)

		expected := map[string]any{
			"operation":      events.CertRenew,
			"serial":         renewed.Certs[0].Serial,
			"revoked_serial": expiring.Serial,
			"thing_id":       tc.thingID,
			"evt_org_id":     tc.orgID,
			"evt_group_id":   tc.groupID,
		}
		msg := fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestCRLFailure(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	// The CRL directory doesn't exist, so certificates get revoked but the CRL can't be written.
	env := newEnv(t, filepath.Join(t.TempDir(), "missing", "crl.pem"))
	svc := env.withEventStore(t)

	issue := func() certs.Cert {
		cert, err := env.svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		return cert
	}
	ctx := env.identityCtx(token)

	cases := []struct {
		desc     string
		replaces bool
		run      func(serial string) (string, error)
		event    func(serial, newSerial string) map[string]any
	}{
		{
			desc: "revoke cert",
			run: func(serial string) (string, error) {
				_, err := svc.RevokeCert(ctx, token, serial)
				return "", err
			},
			event: func(serial, _ string) map[string]any {
				return certEvent(events.CertRevoke, map[string]any{
					"serial":   serial,
					"thing_id": thingID,
				})
			},
		},
		{
			desc:     "rotate cert",
			replaces: true,
			run: func(serial string) (string, error) {
				cert, err := svc.RotateCert(ctx, token, serial, thingID, ttl, keyBits, keyType)
				return cert.Serial, err
			},
			event: func(serial, newSerial string) map[string]any {
				return certEvent(events.CertRotate, map[string]any{
					"serial":         newSerial,
					"revoked_serial": serial,
					"thing_id":       thingID,
				})
			},
		},
		{
			desc:     "renew cert",
			replaces: true,
			run: func(serial string) (string, error) {
				cert, err := svc.RenewCert(ctx, token, serial)
				return cert.Serial, err
			},
			event: func(serial, newSerial string) map[string]any {
				return certEvent(events.CertRenew, map[string]any{
					"serial":         newSerial,
					"revoked_serial": serial,
					"thing_id":       thingID,
				})
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		serial := issue().Serial
		newSerial, err := tc.run(serial)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, certs.ErrFailedCRLGeneration, err)
		assert.True(t, errors.Contains(err, certs.ErrFailedCRLGeneration), msg)
		if tc.replaces {
			assert.NotEmpty(t, newSerial, fmt.Sprintf("%s: the new cert should be returned", tc.desc))
		}

		expected := tc.event(serial, newSerial)
		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}
