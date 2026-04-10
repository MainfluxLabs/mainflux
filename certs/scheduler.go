// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/logger"
	mfcron "github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	renewalWindow     = 30 * 24 * time.Hour
	certRotationTopic = "cert-rotation"
	schedulerEntityID = "cert-rotation-scheduler"
)

// CertEvent is the payload published to the message bus when a certificate is rotated.
type CertEvent struct {
	ThingID    string   `json:"thing_id"`
	ClientCert string   `json:"client_cert"`
	ClientKey  string   `json:"client_key"`
	IssuingCA  string   `json:"issuing_ca"`
	CAChain    []string `json:"ca_chain"`
	Serial     string   `json:"serial"`
	ExpiresAt  string   `json:"expires_at"`
}

// CertScheduler periodically renews certificates approaching expiration
// and notifies Things of their new credentials via the message bus.
type CertScheduler struct {
	repo      Repository
	pki       pki.Agent
	things    protomfx.ThingsServiceClient
	publisher messaging.Publisher
	logger    logger.Logger
	sm        *mfcron.ScheduleManager
}

// NewCertScheduler creates a new certificate rotation scheduler.
func NewCertScheduler(repo Repository, pkiAgent pki.Agent, things protomfx.ThingsServiceClient, pub messaging.Publisher, logger logger.Logger) *CertScheduler {
	return &CertScheduler{
		repo:      repo,
		pki:       pkiAgent,
		things:    things,
		publisher: pub,
		logger:    logger,
		sm:        mfcron.NewScheduleManager(),
	}
}

// Start registers the rotation task with the cron scheduler and blocks until context is cancelled.
func (cs *CertScheduler) Start(ctx context.Context, schedule mfcron.Scheduler) error {
	cs.logger.Info("Certificate rotation scheduler starting")

	task := func() {
		cs.rotateCerts(ctx)
	}

	// Run once immediately on startup.
	task()

	if err := cs.sm.ScheduleRepeatingTask(task, schedule, schedulerEntityID); err != nil {
		return fmt.Errorf("failed to schedule cert rotation: %w", err)
	}

	cs.logger.Info("Certificate rotation scheduler started")

	// Block until context is cancelled.
	<-ctx.Done()
	cs.sm.Stop()
	cs.logger.Info("Certificate rotation scheduler stopped")
	return nil
}

func (cs *CertScheduler) rotateCerts(ctx context.Context) {
	expiring, err := cs.repo.RetrieveExpiring(ctx, renewalWindow)
	if err != nil {
		cs.logger.Error("Failed to retrieve expiring certificates: " + err.Error())
		return
	}

	if len(expiring) == 0 {
		return
	}

	cs.logger.Info(fmt.Sprintf("Found %d expiring certificates, starting rotation", len(expiring)))

	for _, cert := range expiring {
		if err := cs.renewAndNotify(ctx, cert); err != nil {
			cs.logger.Error(fmt.Sprintf("Failed to rotate certificate serial %s for thing %s: %s", cert.Serial, cert.ThingID, err.Error()))
		}
	}
}

func (cs *CertScheduler) renewAndNotify(ctx context.Context, oldCert Cert) error {
	keyType := oldCert.PrivateKeyType
	if keyType == "" {
		keyType = defaultRenewalKeyType
	}

	keyBits := oldCert.KeyBits
	if keyBits == 0 {
		keyBits = defaultRenewalKeyBits
	}

	thingKeyRes, err := cs.things.GetKeyByThingID(ctx, &protomfx.ThingID{Value: oldCert.ThingID})
	if err != nil {
		return errors.Wrap(ErrFailedCertCreation, err)
	}

	pkiCert, err := cs.pki.IssueCert(thingKeyRes.GetValue(), defaultRenewalTTL, keyType, keyBits)
	if err != nil {
		return errors.Wrap(ErrFailedCertCreation, err)
	}

	newCert := Cert{
		ThingID:        oldCert.ThingID,
		ClientCert:     pkiCert.ClientCert,
		IssuingCA:      pkiCert.IssuingCA,
		CAChain:        pkiCert.CAChain,
		ClientKey:      pkiCert.ClientKey,
		PrivateKeyType: pkiCert.PrivateKeyType,
		KeyBits:        pkiCert.KeyBits,
		Serial:         pkiCert.Serial,
		ExpiresAt:      pkiCert.Expire,
	}

	if _, err := cs.repo.Save(ctx, newCert); err != nil {
		return err
	}

	if err := cs.repo.Remove(ctx, oldCert.Serial); err != nil {
		cs.logger.Error(fmt.Sprintf("Failed to revoke old certificate %s after renewal: %s", oldCert.Serial, err.Error()))
	}

	cs.logger.Info(fmt.Sprintf("Certificate rotated for thing %s: old serial %s -> new serial %s", oldCert.ThingID, oldCert.Serial, newCert.Serial))

	return cs.publishCertEvent(newCert)
}

func (cs *CertScheduler) publishCertEvent(cert Cert) error {
	event := CertEvent{
		ThingID:    cert.ThingID,
		ClientCert: cert.ClientCert,
		ClientKey:  cert.ClientKey,
		IssuingCA:  cert.IssuingCA,
		CAChain:    cert.CAChain,
		Serial:     cert.Serial,
		ExpiresAt:  cert.ExpiresAt.Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := protomfx.Message{
		Publisher:   cert.ThingID,
		Protocol:    "certs",
		Payload:     payload,
		ContentType: "application/json",
		Created:     time.Now().UnixNano(),
	}

	subject := nats.GetThingCommandsSubject(cert.ThingID, certRotationTopic)
	return cs.publisher.Publish(subject, msg)
}
