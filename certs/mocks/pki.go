// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

var (
	errPrivateKeyEmpty           = errors.New("private key is empty")
	errPrivateKeyUnsupportedType = errors.New("private key type is unsupported")
)

var (
	// ErrMissingCACert indicates missing ca certificate
	ErrMissingCACert = errors.New("missing ca certificate for certificate signing")

	// ErrFailedCertCreation indicates failed to certificate creation
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation indicates failed certificate revocation
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")
)

type Agent interface {
	// IssueCert is a mock method for Issuing Certificates of PKI Agent.
	IssueCert(cn, ttl, keyType string, keyBits int) (certs.Cert, error)

	// VerifyCert is a mock method for Verifying Certificates of PKI Agent.
	VerifyCert(certPEM string) (*x509.Certificate, error)
}

type agent struct {
	AuthTimeout time.Duration
	TLSCert     tls.Certificate
	X509Cert    *x509.Certificate
	RSABits     int
	TTL         string
	caPEM       string
	mu          sync.Mutex
	counter     uint64
	certs       map[string]certs.Cert
}

func NewPKIAgent(tlsCert tls.Certificate, caCert *x509.Certificate, keyBits int, ttl string, timeout time.Duration) Agent {
	var caPEM string
	if len(tlsCert.Certificate) > 0 {
		caPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: tlsCert.Certificate[0],
		}))
	}

	return &agent{
		AuthTimeout: timeout,
		TLSCert:     tlsCert,
		X509Cert:    caCert,
		RSABits:     keyBits,
		TTL:         ttl,
		caPEM:       caPEM,
		certs:       make(map[string]certs.Cert),
	}
}

func (a *agent) IssueCert(cn, ttl, keyType string, keyBits int) (certs.Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.X509Cert == nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, ErrMissingCACert)
	}

	var priv any
	var err error

	switch keyType {
	case "rsa", "":
		if keyBits == 0 {
			keyBits = 2048
		}
		priv, err = rsa.GenerateKey(rand.Reader, keyBits)
	case "ecdsa", "ec":
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
			return certs.Cert{}, errors.New("unsupported EC key size, use 224, 256, 384, or 521")
		}
		priv, err = ecdsa.GenerateKey(curve, rand.Reader)
	default:
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, errPrivateKeyUnsupportedType)
	}

	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	if ttl == "" {
		ttl = a.TTL
	}

	notBefore := time.Now()
	validFor, err := time.ParseDuration(ttl)
	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
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
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, a.X509Cert, pubKey, a.TLSCert.PrivateKey)
	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	x509cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	var bw, keyOut bytes.Buffer
	buffWriter := bufio.NewWriter(&bw)
	buffKeyOut := bufio.NewWriter(&keyOut)

	if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	buffWriter.Flush()
	certPEM := bw.String()

	block, err := pemBlockForKey(priv)
	if err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	if err := pem.Encode(buffKeyOut, block); err != nil {
		return certs.Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}
	buffKeyOut.Flush()
	keyPEM := keyOut.String()

	cert := certs.Cert{
		ClientCert:     certPEM,
		ClientKey:      keyPEM,
		IssuingCA:      a.caPEM,
		CAChain:        []string{a.caPEM},
		PrivateKeyType: keyType,
		Serial:         x509cert.SerialNumber.String(),
		ExpiresAt:      x509cert.NotAfter,
	}

	a.certs[x509cert.SerialNumber.String()] = cert
	a.counter++

	return cert, nil
}

func (a *agent) VerifyCert(certPEM string) (*x509.Certificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.New("failed to parse certificate"), err)
	}

	_, exists := a.certs[cert.SerialNumber.String()]
	if !exists {
		return nil, errors.New("certificate not found")
	}

	return cert, nil
}

func publicKey(priv any) (any, error) {
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

func pemBlockForKey(priv any) (*pem.Block, error) {
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
