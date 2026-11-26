// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	certs.Service
	things domain.ThingsClient
	pub    events.Publisher
}

// NewEventStoreMiddleware returns a certs service that publishes an event
// for every successful state-changing operation.
func NewEventStoreMiddleware(svc certs.Service, things domain.ThingsClient, pub events.Publisher) certs.Service {
	return eventStore{
		Service: svc,
		things:  things,
		pub:     pub,
	}
}

func (es eventStore) IssueCert(ctx context.Context, token, thingID, ttl string, keyBits int, keyType string) (certs.Cert, error) {
	cert, err := es.Service.IssueCert(ctx, token, thingID, ttl, keyBits, keyType)
	if err != nil {
		return cert, err
	}

	es.publish(ctx, thingID, events.CertIssued{
		Serial:  cert.Serial,
		ThingID: thingID,
	})

	return cert, nil
}

func (es eventStore) RotateCert(ctx context.Context, token, serial, thingID, ttl string, keyBits int, keyType string) (certs.Cert, error) {
	cert, err := es.Service.RotateCert(ctx, token, serial, thingID, ttl, keyBits, keyType)
	if err != nil {
		return cert, err
	}

	es.publish(ctx, thingID, events.CertRotated{
		Serial:        cert.Serial,
		RevokedSerial: serial,
		ThingID:       thingID,
	})

	return cert, nil
}

func (es eventStore) RevokeCert(ctx context.Context, token, serial string) (certs.Revoke, error) {
	// Looked up first because the revoked certificate can't be viewed; a failure only leaves the thing out.
	cert, _ := es.Service.ViewCert(ctx, token, serial)

	revoke, err := es.Service.RevokeCert(ctx, token, serial)
	if err != nil {
		return revoke, err
	}

	es.publish(ctx, cert.ThingID, events.CertRevoked{
		Serial:  serial,
		ThingID: cert.ThingID,
	})

	return revoke, nil
}

func (es eventStore) RenewCert(ctx context.Context, token, serial string) (certs.Cert, error) {
	cert, err := es.Service.RenewCert(ctx, token, serial)
	if err != nil {
		return cert, err
	}

	es.publish(ctx, cert.ThingID, events.CertRenewed{
		Serial:        cert.Serial,
		RevokedSerial: serial,
		ThingID:       cert.ThingID,
	})

	return cert, nil
}

func (es eventStore) publish(ctx context.Context, thingID string, action events.EventAction) {
	var groupID, orgID string
	if thingID != "" {
		groupID, orgID = certs.GetGroupAndOrgIDsByThing(ctx, es.things, thingID)
	}

	es.pub.Publish(ctx, events.Event{
		Action:  action,
		GroupID: groupID,
		OrgID:   orgID,
	})
}
