package pki

import (
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

const (
	rsaKeyType      = "rsa"
	ecdsaKeyType    = "ecdsa"
	ecKeyType       = "ec"
	rsaKeyBlockType = "RSA PRIVATE KEY"
	ecKeyBlockType  = "EC PRIVATE KEY"
	certificateType = "CERTIFICATE"
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
	// IssueCert generates and returns a new certificate.
	IssueCert(cn, ttl, keyType string, keyBits int) (Cert, error)

	// VerifyCert validates that the certificate is valid.
	VerifyCert(certPEM string) (*x509.Certificate, error)
}

var (
	// ErrMissingCACert indicates missing CA certificate.
	ErrMissingCACert = errors.New("missing ca certificate for certificate signing")

	// ErrFailedCertCreation indicates an error in attempting to create a certificate.
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrInvalidCert indicates certificate is invalid.
	ErrInvalidCert = errors.New("certificate is invalid")

	// ErrExpiredCert indicates certificate has expired.
	ErrExpiredCert = errors.New("certificate has expired")

	// ErrPrivateKeyEmpty indicates that PK failed to load.
	ErrPrivateKeyEmpty = errors.New("private key is empty")

	// ErrPrivateKeyUnsupportedType indicates that private key type is not supported.
	ErrPrivateKeyUnsupportedType = errors.New("unsupported key type")

	// ErrFailedCACertParsing indicates certificate failed to parse.
	ErrFailedCACertParsing = errors.New("failed to parse ca certificate")

	// ErrFailedPEMParsing indicates PEM failed to parse.
	ErrFailedPEMParsing = errors.New("failed to parse certificate pem")
)

type agent struct {
	mu     sync.RWMutex
	caCert *x509.Certificate
	caKey  any
	caPEM  string
}

func NewAgent(tlsCert tls.Certificate) (Agent, error) {
	if len(tlsCert.Certificate) == 0 {
		return nil, ErrMissingCACert
	}

	caCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(ErrFailedCACertParsing, err)
	}

	if tlsCert.PrivateKey == nil {
		return nil, ErrPrivateKeyEmpty
	}

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  certificateType,
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

	var privateKey any
	var privKeyPEM string
	var pkType string

	switch strings.ToLower(keyType) {
	case rsaKeyType, "":
		if keyBits == 0 {
			keyBits = 2048
		}
		rsaKey, err := rsa.GenerateKey(rand.Reader, keyBits)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = rsaKey
		pkType = rsaKeyType

		privKeyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  rsaKeyBlockType,
			Bytes: privKeyBytes,
		}))

	case ecKeyType, ecdsaKeyType:
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
			return Cert{}, errors.New("unsupported ec key size, use 224, 256, 384, or 521")
		}

		ecKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = ecKey
		pkType = ecKeyType

		privKeyBytes, err := x509.MarshalECPrivateKey(ecKey)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  ecKeyBlockType,
			Bytes: privKeyBytes,
		}))

	default:
		return Cert{}, ErrPrivateKeyUnsupportedType
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
		return Cert{}, ErrPrivateKeyUnsupportedType
	}

	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	certPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  certificateType,
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
		return nil, ErrFailedPEMParsing
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidCert, err)
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return nil, ErrExpiredCert
	}

	roots := x509.NewCertPool()
	roots.AddCert(a.caCert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return nil, errors.Wrap(ErrInvalidCert, err)
	}

	return cert, nil
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

	return 0, fmt.Errorf("invalid ttl format: %s (use duration like '8760h' or hours as integer)", ttl)
}
