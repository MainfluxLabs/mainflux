// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/certs/mocks"
	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	authmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongValue = "wrong-value"
	email      = "user@example.com"
	token      = email
	password   = "password"
	thingsNum  = 1
	thingKey   = "thingKey"
	thingID    = "1"
	ttl        = "1h"
	keyBits    = 2048
	keyType    = "rsa"
	certNum    = 10

	cfgLogLevel       = "error"
	cfgClientTLS      = false
	cfgServerCert     = ""
	cfgServerKey      = ""
	cfgCertsURL       = "http://localhost"
	cfgJaegerURL      = ""
	cfgAuthURL        = "localhost:8181"
	cfgAuthTimeout    = "1s"
	caPath            = "../docker/ssl/certs/ca.crt"
	caKeyPath         = "../docker/ssl/certs/ca.key"
	cfgSignHoursValid = "24h"
	cfgSignRSABits    = 2048
)

var usersList = []users.User{{Email: email, Password: password}}

func newService() (certs.Service, pki.Agent, error) {
	auth := authmock.NewAuthService("", usersList, nil)
	repo := mocks.NewCertsRepository()

	ths := make(map[string]things.Thing, thingsNum)
	for i := 0; i < thingsNum; i++ {
		id := strconv.Itoa(i + 1)
		ths[id] = things.Thing{
			ID:  id,
			Key: thingKey,
		}
	}
	tc := thmocks.NewThingsServiceClient(map[string]things.Profile{}, ths, map[string]things.Group{})

	tlsCert, caCert, err := loadCertificates(caPath, caKeyPath)
	if err != nil {
		return nil, nil, err
	}

	authTimeout, err := time.ParseDuration(cfgAuthTimeout)
	if err != nil {
		return nil, nil, err
	}

	pkiAgent, err := pki.NewAgent(tlsCert)
	if err != nil {
		return nil, nil, err
	}

	c := certs.Config{
		LogLevel:       cfgLogLevel,
		ClientTLS:      cfgClientTLS,
		ServerCert:     cfgServerCert,
		ServerKey:      cfgServerKey,
		CertsURL:       cfgCertsURL,
		JaegerURL:      cfgJaegerURL,
		AuthURL:        cfgAuthURL,
		SignTLSCert:    tlsCert,
		SignX509Cert:   caCert,
		SignHoursValid: cfgSignHoursValid,
		SignRSABits:    cfgSignRSABits,
		AuthTimeout:    authTimeout,
	}

	return certs.New(auth, tc, repo, c, pkiAgent), pkiAgent, nil
}

func TestIssueCert(t *testing.T) {
	svc, pkiAgent, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))
	require.NotNil(t, pkiAgent, "pki agent should not be nil")

	cases := []struct {
		token   string
		desc    string
		thingID string
		ttl     string
		keyType string
		keyBits int
		err     error
	}{
		{
			desc:    "issue new cert",
			token:   token,
			thingID: thingID,
			ttl:     ttl,
			keyType: keyType,
			keyBits: keyBits,
			err:     nil,
		},
		{
			desc:    "issue new cert for non existing thing id",
			token:   token,
			thingID: "2",
			ttl:     ttl,
			keyType: keyType,
			keyBits: keyBits,
			err:     certs.ErrFailedCertCreation,
		},
		{
			desc:    "issue new cert with invalid token",
			token:   wrongValue,
			thingID: thingID,
			ttl:     ttl,
			keyType: keyType,
			keyBits: keyBits,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "issue new cert with invalid key bits",
			token:   token,
			thingID: thingID,
			ttl:     ttl,
			keyType: keyType,
			keyBits: -2,
			err:     certs.ErrFailedCertCreation,
		},
	}

	for _, tc := range cases {
		c, err := svc.IssueCert(context.Background(), tc.token, tc.thingID, tc.ttl, tc.keyBits, tc.keyType)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, c.ClientCert, fmt.Sprintf("%s: client cert should not be empty", tc.desc))
			assert.NotEmpty(t, c.ClientKey, fmt.Sprintf("%s: client key should not be empty", tc.desc))
			assert.NotEmpty(t, c.Serial, fmt.Sprintf("%s: serial should not be empty", tc.desc))
			assert.Equal(t, tc.thingID, c.ThingID, fmt.Sprintf("%s: thing mismatch", tc.desc))

			cert, _ := readCert([]byte(c.ClientCert))
			if cert != nil {
				assert.True(t, strings.Contains(cert.Subject.CommonName, thingKey),
					fmt.Sprintf("%s: expected cert cn to contain thing key", tc.desc))
			}

			_, err := pkiAgent.VerifyCert(c.ClientCert)
			assert.NoError(t, err, fmt.Sprintf("%s: certificate verification failed", tc.desc))
		}
	}
}

func TestRevokeCert(t *testing.T) {
	svc, _, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	issuedCert, err := svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))

	cases := []struct {
		token    string
		desc     string
		serialID string
		err      error
	}{
		{
			desc:     "revoke cert",
			token:    token,
			serialID: issuedCert.Serial,
			err:      nil,
		},
		{
			desc:     "revoke cert with invalid token",
			token:    wrongValue,
			serialID: issuedCert.Serial,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "revoke cert for invalid serial id",
			token:    token,
			serialID: "invalid-serial",
			err:      certs.ErrFailedCertRevocation,
		},
	}

	for _, tc := range cases {
		revoke, err := svc.RevokeCert(context.Background(), tc.token, tc.serialID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.False(t, revoke.RevocationTime.IsZero(), fmt.Sprintf("%s: revocation time should not be zero", tc.desc))
		}
	}
}

func TestListCerts(t *testing.T) {
	svc, _, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	for i := 0; i < certNum; i++ {
		_, err = svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
		require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))
	}

	cases := []struct {
		token   string
		desc    string
		thingID string
		offset  uint64
		limit   uint64
		size    uint64
		err     error
	}{
		{
			desc:    "list all certs with valid token",
			token:   token,
			thingID: thingID,
			offset:  0,
			limit:   certNum,
			size:    certNum,
			err:     nil,
		},
		{
			desc:    "list all certs with invalid token",
			token:   wrongValue,
			thingID: thingID,
			offset:  0,
			limit:   certNum,
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list half certs with valid token",
			token:   token,
			thingID: thingID,
			offset:  certNum / 2,
			limit:   certNum,
			size:    certNum / 2,
			err:     nil,
		},
		{
			desc:    "list last cert with valid token",
			token:   token,
			thingID: thingID,
			offset:  certNum - 1,
			limit:   certNum,
			size:    1,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListCerts(context.Background(), tc.token, tc.thingID, tc.offset, tc.limit)
		size := uint64(len(page.Certs))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListSerials(t *testing.T) {
	svc, _, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	var issuedSerials []string
	for i := 0; i < certNum; i++ {
		cert, err := svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
		require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))
		issuedSerials = append(issuedSerials, cert.Serial)
	}

	cases := []struct {
		token   string
		desc    string
		thingID string
		offset  uint64
		limit   uint64
		size    uint64
		err     error
	}{
		{
			desc:    "list all serials with valid token",
			token:   token,
			thingID: thingID,
			offset:  0,
			limit:   certNum,
			size:    certNum,
			err:     nil,
		},
		{
			desc:    "list all serials with invalid token",
			token:   wrongValue,
			thingID: thingID,
			offset:  0,
			limit:   certNum,
			size:    0,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list half serials with valid token",
			token:   token,
			thingID: thingID,
			offset:  certNum / 2,
			limit:   certNum,
			size:    certNum / 2,
			err:     nil,
		},
		{
			desc:    "list last serial with valid token",
			token:   token,
			thingID: thingID,
			offset:  certNum - 1,
			limit:   certNum,
			size:    1,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListSerials(context.Background(), tc.token, tc.thingID, tc.offset, tc.limit)
		size := uint64(len(page.Certs))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil && size > 0 {
			for i, cert := range page.Certs {
				expectedIndex := int(tc.offset) + i
				if expectedIndex < len(issuedSerials) {
					assert.Equal(t, issuedSerials[expectedIndex], cert.Serial,
						fmt.Sprintf("%s: serial mismatch at index %d", tc.desc, i))
				}
			}
		}
	}
}

func TestViewCert(t *testing.T) {
	svc, _, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	issuedCert, err := svc.IssueCert(context.Background(), token, thingID, ttl, keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))

	cases := []struct {
		token    string
		desc     string
		serialID string
		err      error
	}{
		{
			desc:     "view cert with valid token and serial",
			token:    token,
			serialID: issuedCert.Serial,
			err:      nil,
		},
		{
			desc:     "view cert with invalid token",
			token:    wrongValue,
			serialID: issuedCert.Serial,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "view cert with invalid serial",
			token:    token,
			serialID: wrongValue,
			err:      dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		cert, err := svc.ViewCert(context.Background(), tc.token, tc.serialID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, issuedCert.Serial, cert.Serial, fmt.Sprintf("%s: serial mismatch", tc.desc))
			assert.Equal(t, issuedCert.ThingID, cert.ThingID, fmt.Sprintf("%s: thing ID mismatch", tc.desc))
			assert.Equal(t, issuedCert.ClientCert, cert.ClientCert, fmt.Sprintf("%s: client cert mismatch", tc.desc))
			assert.Equal(t, issuedCert.ExpiresAt, cert.ExpiresAt, fmt.Sprintf("%s: expiration mismatch", tc.desc))
		}
	}
}

func TestRenewCert(t *testing.T) {
	svc, _, err := newService()
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	issuedCert, err := svc.IssueCert(context.Background(), token, thingID, "1h", keyBits, keyType)
	require.Nil(t, err, fmt.Sprintf("unexpected cert creation error: %s\n", err))

	cases := []struct {
		token    string
		desc     string
		serialID string
		err      error
	}{
		{
			desc:     "renew cert with valid token and serial",
			token:    token,
			serialID: issuedCert.Serial,
			err:      nil,
		},
		{
			desc:     "renew cert with invalid token",
			token:    wrongValue,
			serialID: issuedCert.Serial,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "renew cert with invalid serial",
			token:    token,
			serialID: wrongValue,
			err:      dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		renewedCert, err := svc.RenewCert(context.Background(), tc.token, tc.serialID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEqual(t, issuedCert.Serial, renewedCert.Serial, fmt.Sprintf("%s: renewed cert should have different serial", tc.desc))
			assert.Equal(t, issuedCert.ThingID, renewedCert.ThingID, fmt.Sprintf("%s: thing should match", tc.desc))
			assert.NotEmpty(t, renewedCert.ClientCert, fmt.Sprintf("%s: client cert should not be empty", tc.desc))
			assert.NotEmpty(t, renewedCert.ClientKey, fmt.Sprintf("%s: client key should not be empty", tc.desc))
			assert.True(t, renewedCert.ExpiresAt.After(issuedCert.ExpiresAt), fmt.Sprintf("%s: renewed cert should expire later", tc.desc))
		}
	}
}

func loadCertificates(caPath, caKeyPath string) (tls.Certificate, *x509.Certificate, error) {
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mainflux Test CA"},
			CommonName:   "Mainflux Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{caCertDER},
		PrivateKey:  caPrivKey,
	}

	return tlsCert, caCert, nil
}

func readCert(b []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode pem data")
	}

	return x509.ParseCertificate(block.Bytes)
}
