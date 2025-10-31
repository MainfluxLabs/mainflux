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
	mu             sync.Mutex
	counter        uint64
	certsBySerial  map[string]certs.Cert
	certsByThingID map[string]map[string][]certs.Cert
	revokedCerts   map[string]certs.RevokedCert
}

func NewCertsRepository() certs.Repository {
	return &certsRepoMock{
		certsBySerial:  make(map[string]certs.Cert),
		certsByThingID: make(map[string]map[string][]certs.Cert),
		revokedCerts:   make(map[string]certs.RevokedCert),
	}
}

func (c *certsRepoMock) Save(ctx context.Context, cert certs.Cert) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt := certs.Cert{
		OwnerID:        cert.OwnerID,
		ThingID:        cert.ThingID,
		Serial:         cert.Serial,
		Expire:         cert.Expire,
		ClientCert:     cert.ClientCert,
		ClientKey:      cert.ClientKey,
		IssuingCA:      cert.IssuingCA,
		CAChain:        cert.CAChain,
		PrivateKeyType: cert.PrivateKeyType,
	}

	if _, ok := c.certsByThingID[cert.OwnerID]; !ok {
		c.certsByThingID[cert.OwnerID] = make(map[string][]certs.Cert)
	}

	if _, ok := c.certsByThingID[cert.OwnerID][cert.ThingID]; !ok {
		c.certsByThingID[cert.OwnerID][cert.ThingID] = []certs.Cert{}
	}

	c.certsByThingID[cert.OwnerID][cert.ThingID] = append(c.certsByThingID[cert.OwnerID][cert.ThingID], crt)

	c.certsBySerial[cert.Serial] = crt
	c.counter++
	return cert.Serial, nil
}

func (c *certsRepoMock) RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if limit <= 0 {
		return certs.Page{}, nil
	}

	ownerCerts, ok := c.certsByThingID[ownerID]
	if !ok {
		return certs.Page{}, dbutil.ErrNotFound
	}

	var allCerts []certs.Cert
	for _, thingCerts := range ownerCerts {
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

// Remove removes certificate from DB by serial ID (matches postgres implementation)
func (c *certsRepoMock) Remove(ctx context.Context, ownerID, serialID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// First retrieve the cert to get thingID
	cert, ok := c.certsBySerial[serialID]
	if !ok {
		return dbutil.ErrNotFound
	}

	// Verify ownership
	if cert.OwnerID != ownerID {
		return dbutil.ErrNotFound
	}

	thingID := cert.ThingID

	// Remove from certsBySerial
	delete(c.certsBySerial, serialID)

	// Add to revoked certs
	c.revokedCerts[serialID] = certs.RevokedCert{
		Serial:    serialID,
		OwnerID:   ownerID,
		ThingID:   thingID,
		RevokedAt: time.Now(),
	}

	// Remove from certsByThingID
	if thingCerts, ok := c.certsByThingID[ownerID][thingID]; ok {
		// Find and remove the specific cert
		for i, tc := range thingCerts {
			if tc.Serial == serialID {
				// Remove this cert from the slice
				c.certsByThingID[ownerID][thingID] = append(thingCerts[:i], thingCerts[i+1:]...)
				break
			}
		}

		// Clean up empty maps
		if len(c.certsByThingID[ownerID][thingID]) == 0 {
			delete(c.certsByThingID[ownerID], thingID)
		}

		if len(c.certsByThingID[ownerID]) == 0 {
			delete(c.certsByThingID, ownerID)
		}
	}

	return nil
}

func (c *certsRepoMock) RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if limit <= 0 {
		return certs.Page{}, nil
	}

	thingCerts, ok := c.certsByThingID[ownerID][thingID]
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

func (c *certsRepoMock) RetrieveBySerial(ctx context.Context, ownerID, serialID string) (certs.Cert, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt, ok := c.certsBySerial[serialID]
	if !ok {
		return certs.Cert{}, dbutil.ErrNotFound
	}

	if crt.OwnerID != ownerID {
		return certs.Cert{}, dbutil.ErrNotFound
	}

	return crt, nil
}

func (c *certsRepoMock) RetrieveRevokedCertificates(ctx context.Context) ([]certs.RevokedCert, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	revokedCerts := make([]certs.RevokedCert, 0, len(c.revokedCerts))
	for _, cert := range c.revokedCerts {
		revokedCerts = append(revokedCerts, cert)
	}

	return revokedCerts, nil
}

