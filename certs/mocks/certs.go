// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ certs.Repository = (*certsRepoMock)(nil)

type certsRepoMock struct {
	mu            sync.Mutex
	counter       uint64
	certsBySerial map[string]certs.Cert
	certsByThing  map[string][]certs.Cert
	revokedCerts  map[string]certs.RevokedCert
}

func NewCertsRepository() certs.Repository {
	return &certsRepoMock{
		certsBySerial: make(map[string]certs.Cert),
		certsByThing:  make(map[string][]certs.Cert),
		revokedCerts:  make(map[string]certs.RevokedCert),
	}
}

func (c *certsRepoMock) Save(ctx context.Context, cert certs.Cert) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt := certs.Cert{
		ThingID:        cert.ThingID,
		Serial:         cert.Serial,
		ExpiresAt:      cert.ExpiresAt,
		ClientCert:     cert.ClientCert,
		ClientKey:      cert.ClientKey,
		IssuingCA:      cert.IssuingCA,
		CAChain:        cert.CAChain,
		PrivateKeyType: cert.PrivateKeyType,
		KeyBits:        cert.KeyBits,
	}

	if _, ok := c.certsByThing[cert.ThingID]; !ok {
		c.certsByThing[cert.ThingID] = []certs.Cert{}
	}

	c.certsByThing[cert.ThingID] = append(c.certsByThing[cert.ThingID], crt)
	c.certsBySerial[cert.Serial] = crt
	c.counter++

	return cert.Serial, nil
}

func (c *certsRepoMock) RetrieveAll(ctx context.Context, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if limit <= 0 {
		return certs.Page{}, nil
	}

	// Collect all certificates from all things
	var allCerts []certs.Cert
	for _, thingCerts := range c.certsByThing {
		allCerts = append(allCerts, thingCerts...)
	}

	start := offset
	if start > uint64(len(allCerts)) {
		start = uint64(len(allCerts))
	}

	end := start + limit
	if end > uint64(len(allCerts)) {
		end = uint64(len(allCerts))
	}

	page := certs.Page{
		Certs: allCerts[start:end],
		Total: uint64(len(allCerts)),
	}

	return page, nil
}

func (c *certsRepoMock) Remove(ctx context.Context, serial string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cert, ok := c.certsBySerial[serial]
	if !ok {
		return dbutil.ErrNotFound
	}

	thingID := cert.ThingID
	delete(c.certsBySerial, serial)

	c.revokedCerts[serial] = certs.RevokedCert{
		Serial:    serial,
		ThingID:   thingID,
		RevokedAt: time.Now(),
	}

	if thingCerts, ok := c.certsByThing[thingID]; ok {
		for i, tc := range thingCerts {
			if tc.Serial == serial {
				c.certsByThing[thingID] = append(thingCerts[:i], thingCerts[i+1:]...)
				break
			}
		}
		if len(c.certsByThing[thingID]) == 0 {
			delete(c.certsByThing, thingID)
		}
	}

	return nil
}

func (c *certsRepoMock) RetrieveByThing(ctx context.Context, thingID string, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if limit <= 0 {
		return certs.Page{}, nil
	}

	thingCerts, ok := c.certsByThing[thingID]
	if !ok {
		return certs.Page{}, dbutil.ErrNotFound
	}

	start := offset
	if start > uint64(len(thingCerts)) {
		start = uint64(len(thingCerts))
	}

	end := start + limit
	if end > uint64(len(thingCerts)) {
		end = uint64(len(thingCerts))
	}

	page := certs.Page{
		Certs: thingCerts[start:end],
		Total: uint64(len(thingCerts)),
	}

	return page, nil
}

func (c *certsRepoMock) RetrieveBySerial(ctx context.Context, serial string) (certs.Cert, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt, ok := c.certsBySerial[serial]
	if !ok {
		return certs.Cert{}, dbutil.ErrNotFound
	}

	return crt, nil
}

func (c *certsRepoMock) RetrieveRevokedCerts(ctx context.Context) ([]certs.RevokedCert, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	revokedCerts := make([]certs.RevokedCert, 0, len(c.revokedCerts))
	for _, cert := range c.revokedCerts {
		revokedCerts = append(revokedCerts, cert)
	}

	return revokedCerts, nil
}
