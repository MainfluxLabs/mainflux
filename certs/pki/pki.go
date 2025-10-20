// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
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
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

type Cert struct {
	ClientCert     string    `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string    `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	Serial         string    `json:"serial" mapstructure:"serial_number"`
	Expire         time.Time `json:"expire" mapstructure:"-"`
}

type Agent interface {
	IssueCert(cn, ttl, keyType string, keyBits int) (Cert, error)
	VerifyCert(certPEM string) (*x509.Certificate, error)
	CreateCRL(revokedCerts []pkix.RevokedCertificate) ([]byte, error)
}

var (
	// ErrMissingCACertificate indicates missing CA certificate
	ErrMissingCACertificate = errors.New("missing CA certificate for certificate signing")
	// ErrFailedCertCreation indicates failed to certificate creation
	ErrFailedCertCreation = errors.New("failed to create client certificate")
	// ErrFailedCertRevocation indicates failed certificate revocation
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")
	// ErrCertificateInvalid indicates certificate is invalid
	ErrCertificateInvalid = errors.New("certificate is invalid")
	// ErrCertificateExpired indicates certificate has expired
	ErrCertificateExpired = errors.New("certificate has expired")
	// ErrCertificateRevoked indicates certificate has been revoked
	ErrCertificateRevoked = errors.New("certificate has been revoked")

	errPrivateKeyEmpty           = errors.New("private key is empty")
	errPrivateKeyUnsupportedType = errors.New("private key type is unsupported")
)

type agent struct {
	mu     sync.RWMutex
	caCert *x509.Certificate
	caKey  interface{}
	caPEM  string
}

func NewAgent(caCertPEM, caKeyPEM string) (Agent, error) {
	if caCertPEM == "" || caKeyPEM == "" {
		return nil, ErrMissingCACertificate
	}

	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return nil, errors.New("failed to parse CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.New("failed to parse CA certificate"), err)
	}

	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return nil, errors.New("failed to parse CA private key PEM")
	}

	var caKey interface{}
	switch caKeyBlock.Type {
	case "RSA PRIVATE KEY":
		caKey, err = x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	case "EC PRIVATE KEY":
		caKey, err = x509.ParseECPrivateKey(caKeyBlock.Bytes)
	case "PRIVATE KEY":
		caKey, err = x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	default:
		return nil, errors.New("unsupported CA private key type: " + caKeyBlock.Type)
	}

	if err != nil {
		return nil, errors.Wrap(errors.New("failed to parse CA private key"), err)
	}

	return &agent{
		caCert: caCert,
		caKey:  caKey,
		caPEM:  string(caCertPEM),
	}, nil
}

func NewAgentFromTLS(tlsCert tls.Certificate) (Agent, error) {
	if len(tlsCert.Certificate) == 0 {
		return nil, ErrMissingCACertificate
	}

	caCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(errors.New("failed to parse CA certificate"), err)
	}

	if tlsCert.PrivateKey == nil {
		return nil, errPrivateKeyEmpty
	}

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tlsCert.Certificate[0],
	})

	return &agent{
		caCert: caCert,
		caKey:  tlsCert.PrivateKey,
		caPEM:  string(caPEM),
	}, nil
}

func (a *agent) IssueCert(cn, ttl, keyType string, keyBits int) (Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	duration, err := parseTTL(ttl)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	var privateKey interface{}
	var privKeyPEM string
	var pkType string

	switch strings.ToLower(keyType) {
	case "rsa", "":
		if keyBits == 0 {
			keyBits = 2048
		}
		rsaKey, err := rsa.GenerateKey(rand.Reader, keyBits)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = rsaKey
		pkType = "rsa"

		privKeyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privKeyBytes,
		}))

	case "ec", "ecdsa":
		var curve elliptic.Curve
		switch keyBits {
		case 224:
			curve = elliptic.P224()
		case 256, 0:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			return Cert{}, errors.New("unsupported EC key size, use 224, 256, 384, or 521")
		}

		ecKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = ecKey
		pkType = "ec"

		privKeyBytes, err := x509.MarshalECPrivateKey(ecKey)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privKeyBytes,
		}))

	default:
		return Cert{}, errPrivateKeyUnsupportedType
	}

	serialNumber, err := generateSerialNumber()
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(duration)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	var certDER []byte
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		certDER, err = x509.CreateCertificate(rand.Reader, &template, a.caCert, &key.PublicKey, a.caKey)
	case *ecdsa.PrivateKey:
		certDER, err = x509.CreateCertificate(rand.Reader, &template, a.caCert, &key.PublicKey, a.caKey)
	default:
		return Cert{}, errPrivateKeyUnsupportedType
	}

	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	certPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	caChain := []string{a.caPEM}

	cert := Cert{
		ClientCert:     certPEM,
		IssuingCA:      a.caPEM,
		CAChain:        caChain,
		ClientKey:      privKeyPEM,
		PrivateKeyType: pkType,
		Serial:         serialNumber.String(),
		Expire:         notAfter,
	}

	return cert, nil
}

func (a *agent) VerifyCert(certPEM string) (*x509.Certificate, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(ErrCertificateInvalid, err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, ErrCertificateExpired
	}

	roots := x509.NewCertPool()
	roots.AddCert(a.caCert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return nil, errors.Wrap(ErrCertificateInvalid, err)
	}

	return cert, nil
}

func (a *agent) CreateCRL(revokedCerts []pkix.RevokedCertificate) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()
	crlTemplate := &x509.RevocationList{
		Number:              big.NewInt(1),
		ThisUpdate:          now,
		NextUpdate:          now.Add(24 * time.Hour),
		RevokedCertificates: revokedCerts,
	}

	signer, ok := a.caKey.(crypto.Signer)
	if !ok {
		return nil, errors.New("CA private key does not implement crypto.Signer")
	}

	crlBytes, err := x509.CreateRevocationList(rand.Reader, crlTemplate, a.caCert, signer)
	if err != nil {
		return nil, errors.Wrap(errors.New("failed to create CRL"), err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crlBytes,
	}), nil
}

func generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	return serialNumber, nil
}

func parseTTL(ttl string) (time.Duration, error) {
	if ttl == "" {
		return 8760 * time.Hour, nil
	}

	duration, err := time.ParseDuration(ttl)
	if err == nil {
		return duration, nil
	}

	hours, err := strconv.Atoi(ttl)
	if err == nil {
		return time.Duration(hours) * time.Hour, nil
	}

	return 0, fmt.Errorf("invalid TTL format: %s (use duration like '8760h' or hours as integer)", ttl)
}
