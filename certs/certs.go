// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"time"
)

// ConfigsPage contains page related metadata as well as list
type Page struct {
	Total uint64
	Certs []Cert
}

type RevokedCert struct {
	Serial    string
	RevokedAt time.Time
	ThingID   string
	OwnerID   string
}

// Repository specifies a Config persistence API.
type Repository interface {
	// Save  saves cert for thing into database
	Save(ctx context.Context, cert Cert) (string, error)

	// RetrieveAll retrieve issued certificates for given owner ID
	RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (Page, error)

	// Remove removes certificate from DB for a given thing ID
	Remove(ctx context.Context, ownerID, thingID string) error

	// RetrieveByThing retrieves issued certificates for a given thing ID
	RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (Page, error)

	// RetrieveBySerial retrieves a certificate for a given serial ID
	RetrieveBySerial(ctx context.Context, ownerID, serialID string) (Cert, error)

	// Retrieves the serials of all revoked certificates
	RetrieveRevokedCertificates(ctx context.Context) ([]RevokedCert, error)
}
