// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/MainfluxLabs/mainflux/certs/pki"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	// ErrFailedCertCreation failed to create certificate
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation failed to revoke certificate
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")

	// ErrNotEligibleForRenewal indicates the certificate cannot be renewed yet because it expires more than 30 days from now.
	ErrNotEligibleForRenewal = errors.New("certificate not eligible for renewal yet")

	errFailedToRemoveCertFromDB = errors.New("failed to remove cert serial from db")

	errFailedCRLGeneration = errors.New("failed to generate CRL")
)

const (
	defaultRenewalTTL     = "8760h"
	defaultRenewalKeyType = "rsa"
	defaultRenewalKeyBits = 2048
)

var _ Service = (*certsService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// IssueCert issues certificate for given thing id if access is granted with token.
	IssueCert(ctx context.Context, token, thingID, ttl string, keyBits int, keyType string) (Cert, error)

	// ListCerts lists certificates issued for a given thing ID.
	ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error)

	// ListSerials lists certificate serial numbers issued for a given thing ID.
	ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error)

	// ViewCert retrieves the certificate issued for a given serial ID.
	ViewCert(ctx context.Context, token, serial string) (Cert, error)

	// RevokeCert revokes a certificate for a given serial ID.
	RevokeCert(ctx context.Context, token, serial string) (Revoke, error)

	// RenewCert extends the expiration date of a certificate.
	RenewCert(ctx context.Context, token, serial string) (Cert, error)
}

// Config defines the service parameters
type Config struct {
	LogLevel       string
	ClientTLS      bool
	CaCerts        string
	HTTPPort       string
	ServerCert     string
	ServerKey      string
	CertsURL       string
	JaegerURL      string
	AuthURL        string
	AuthTimeout    time.Duration
	SignTLSCert    tls.Certificate
	SignX509Cert   *x509.Certificate
	SignRSABits    int
	SignHoursValid string
	CRLPath        string
}

type certsService struct {
	auth      domain.AuthClient
	things    domain.ThingsClient
	certsRepo Repository
	conf      Config
	pki       pki.Agent
}

// New returns new Certs service.
func New(auth domain.AuthClient, things domain.ThingsClient, certs Repository, config Config, pkiAgent pki.Agent) Service {
	return &certsService{
		certsRepo: certs,
		things:    things,
		auth:      auth,
		conf:      config,
		pki:       pkiAgent,
	}
}

// Revoke defines the conditions to revoke a certificate
type Revoke struct {
	RevocationTime time.Time `mapstructure:"revocation_time"`
}

// Cert defines the certificate parameters
type Cert struct {
	ThingID        string    `json:"thing_id" mapstructure:"thing_id"`
	ClientCert     string    `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string    `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	KeyBits        int       `json:"key_bits" mapstructure:"key_bits"`
	Serial         string    `json:"serial" mapstructure:"serial_number"`
	ExpiresAt      time.Time `json:"expires_at" mapstructure:"-"`
}

func (cs *certsService) IssueCert(ctx context.Context, token, thingID string, ttl string, keyBits int, keyType string) (Cert, error) {
	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return Cert{}, err
	}

	thingKey, err := cs.things.GetKeyByThingID(ctx, thingID)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	pkiCert, err := cs.pki.IssueCert(thingKey.Value, ttl, keyType, keyBits)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	c := Cert{
		ThingID:        thingID,
		ClientCert:     pkiCert.ClientCert,
		IssuingCA:      pkiCert.IssuingCA,
		CAChain:        pkiCert.CAChain,
		ClientKey:      pkiCert.ClientKey,
		PrivateKeyType: pkiCert.PrivateKeyType,
		KeyBits:        pkiCert.KeyBits,
		Serial:         pkiCert.Serial,
		ExpiresAt:      pkiCert.Expire,
	}

	_, err = cs.certsRepo.Save(ctx, c)
	if err != nil {
		return Cert{}, err
	}

	return c, nil
}

func (cs *certsService) RevokeCert(ctx context.Context, token, serial string) (Revoke, error) {
	var revoke Revoke

	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return revoke, err
	}

	_, err = cs.certsRepo.RetrieveBySerial(ctx, serial)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	if err = cs.certsRepo.Remove(ctx, serial); err != nil {
		return revoke, errors.Wrap(errFailedToRemoveCertFromDB, err)
	}

	if err = cs.regenerateCRL(ctx); err != nil {
		return revoke, errors.Wrap(errFailedCRLGeneration, err)
	}

	revoke.RevocationTime = time.Now()
	return revoke, nil
}

func (cs *certsService) regenerateCRL(ctx context.Context) error {
	if cs.conf.CRLPath == "" || cs.pki == nil {
		return nil
	}
	return GenerateCRLFile(ctx, cs.certsRepo, cs.pki, cs.conf.CRLPath)
}

// GenerateCRLFile generates a PEM-encoded CRL file from the current revoked
// certificates and writes it to the given path. It is used both at startup
// and after each revocation.
func GenerateCRLFile(ctx context.Context, repo Repository, pkiAgent pki.Agent, crlPath string) error {
	revokedCerts, err := repo.RetrieveRevokedCerts(ctx)
	if err != nil {
		return err
	}

	revokedSerials := make([]pki.RevokedSerial, len(revokedCerts))
	for i, rc := range revokedCerts {
		revokedSerials[i] = pki.RevokedSerial{
			Serial:    rc.Serial,
			RevokedAt: rc.RevokedAt,
		}
	}

	crlPEM, err := pkiAgent.GenerateCRL(revokedSerials)
	if err != nil {
		return err
	}

	return os.WriteFile(crlPath, crlPEM, 0644)
}

func (cs *certsService) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error) {
	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return Page{}, err
	}

	cp, err := cs.certsRepo.RetrieveByThing(ctx, thingID, offset, limit)
	if err != nil {
		return Page{}, err
	}

	return cp, nil
}

func (cs *certsService) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error) {
	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return Page{}, err
	}

	return cs.certsRepo.RetrieveByThing(ctx, thingID, offset, limit)
}

func (cs *certsService) ViewCert(ctx context.Context, token, serial string) (Cert, error) {
	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return Cert{}, err
	}

	cert, err := cs.certsRepo.RetrieveBySerial(ctx, serial)
	if err != nil {
		return Cert{}, err
	}

	return cert, nil
}

func (cs *certsService) RenewCert(ctx context.Context, token, serial string) (Cert, error) {
	_, err := cs.auth.Identify(ctx, token)
	if err != nil {
		return Cert{}, err
	}

	oldCert, err := cs.certsRepo.RetrieveBySerial(ctx, serial)
	if err != nil {
		return Cert{}, err
	}

	if time.Until(oldCert.ExpiresAt) > 30*24*time.Hour {
		return Cert{}, ErrNotEligibleForRenewal
	}

	// Preserve the original certificate's key type and bits during renewal.
	// Fall back to defaults if the original values are not available.
	keyType := oldCert.PrivateKeyType
	if keyType == "" {
		keyType = defaultRenewalKeyType
	}

	keyBits := oldCert.KeyBits
	if keyBits == 0 {
		keyBits = defaultRenewalKeyBits
	}

	return cs.IssueCert(ctx, token, oldCert.ThingID, defaultRenewalTTL, keyBits, keyType)
}
