// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	errPrivateKeyEmpty           = errors.New("private key is empty")
	errPrivateKeyUnsupportedType = errors.New("private key type is unsupported")
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

var (
	// ErrMissingCACertificate indicates missing CA certificate
	ErrMissingCACertificate = errors.New("missing CA certificate for certificate signing")

	// ErrFailedCertCreation indicates failed to certificate creation
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation indicates failed certificate revocation
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")

	errFailedVaultCertIssue = errors.New("failed to issue vault certificate")
	errFailedVaultRead      = errors.New("failed to read vault certificate")
	errFailedCertDecoding   = errors.New("failed to decode response from vault service")
)

type Agent interface {
	// IssueCert issues certificate on PKI
	IssueCert(cn string, ttl, keyType string, keyBits int) (Cert, error)

	// Read retrieves certificate from PKI
	Read(serial string) (Cert, error)

	// Revoke revokes certificate from PKI
	Revoke(serial string) (time.Time, error)
}

type agent struct {
	AuthTimeout time.Duration
	TLSCert     tls.Certificate
	X509Cert    *x509.Certificate
	RSABits     int
	TTL         string
	mu          sync.Mutex
	counter     uint64
	certs       map[string]Cert
}

func NewPkiAgent(tlsCert tls.Certificate, caCert *x509.Certificate, keyBits int, ttl string, timeout time.Duration) Agent {
	return &agent{
		AuthTimeout: timeout,
		TLSCert:     tlsCert,
		X509Cert:    caCert,
		RSABits:     keyBits,
		TTL:         ttl,
		certs:       make(map[string]Cert),
	}
}

func (a *agent) IssueCert(cn string, ttl, keyType string, keyBits int) (Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.X509Cert == nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, ErrMissingCACertificate)
	}

	var priv interface{}
	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	if ttl == "" {
		ttl = a.TTL
	}

	notBefore := time.Now()
	validFor, err := time.ParseDuration(ttl)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Mainflux"},
			CommonName:         cn,
			OrganizationalUnit: []string{"mainflux"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
	}

	pubKey, err := publicKey(priv)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, a.X509Cert, pubKey, a.TLSCert.PrivateKey)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	x509cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	var bw, keyOut bytes.Buffer
	buffWriter := bufio.NewWriter(&bw)
	buffKeyOut := bufio.NewWriter(&keyOut)

	if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	buffWriter.Flush()
	cert := bw.String()

	block, err := pemBlockForKey(priv)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	if err := pem.Encode(buffKeyOut, block); err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	buffKeyOut.Flush()
	key := keyOut.String()

	a.certs[x509cert.SerialNumber.String()] = Cert{
		ClientCert: cert,
	}
	a.counter++

	return Cert{
		ClientCert: cert,
		ClientKey:  key,
		Serial:     x509cert.SerialNumber.String(),
		Expire:     x509cert.NotAfter,
		IssuingCA:  x509cert.Issuer.String(),
	}, nil
}

func (a *agent) Read(serial string) (Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	crt, ok := a.certs[serial]
	if !ok {
		return Cert{}, dbutil.ErrNotFound
	}

	return crt, nil
}

func (a *agent) Revoke(serial string) (time.Time, error) {
	return time.Now(), nil
}

func publicKey(priv interface{}) (interface{}, error) {
	if priv == nil {
		return nil, errPrivateKeyEmpty
	}
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	default:
		return nil, errPrivateKeyUnsupportedType
	}
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, nil
	}
}
