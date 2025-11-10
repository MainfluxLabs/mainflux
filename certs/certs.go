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
	Serial    string    `db:"serial"`
	ThingID   string    `db:"thing_id"`
	RevokedAt time.Time `db:"revoked_at"`
}

// Repository specifies a Config persistence API.
type Repository interface {
	// Save  saves cert for thing into database
	Save(ctx context.Context, cert Cert) (string, error)

	// RetrieveAll retrieve issued certificates
	RetrieveAll(ctx context.Context, offset, limit uint64) (Page, error)

	// Remove removes certificate from DB for a given serial ID
	Remove(ctx context.Context, serialID string) error

	// RetrieveByThing retrieves issued certificates for a given thing ID
	RetrieveByThing(ctx context.Context, thingID string, offset, limit uint64) (Page, error)

	// RetrieveBySerial retrieves a certificate for a given serial ID
	RetrieveBySerial(ctx context.Context, serialID string) (Cert, error)

	// RetrieveRevokedCerts retrieves all revoked certificates
	RetrieveRevokedCerts(ctx context.Context) ([]RevokedCert, error)
}
