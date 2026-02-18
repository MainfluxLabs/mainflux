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
	RSAKeyType      = "rsa"
	ECDSAKeyType    = "ecdsa"
	ECKeyType       = "ec"
	rsaKeyBlockType = "RSA PRIVATE KEY"
	ecKeyBlockType  = "EC PRIVATE KEY"
	certificateType = "CERTIFICATE"

	DefaultRSAKeyBits   = 2048
	DefaultECDSAKeyBits = 256
)

// ValidRSAKeySizes contains the set of accepted RSA key sizes.
var ValidRSAKeySizes = map[int]bool{
	2048: true,
	4096: true,
}

// ValidECDSAKeySizes contains the set of accepted ECDSA curve sizes.
var ValidECDSAKeySizes = map[int]bool{
	224: true,
	256: true,
	384: true,
	521: true,
}

type Cert struct {
	ClientCert     string    `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string    `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	KeyBits        int       `json:"key_bits" mapstructure:"key_bits"`
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

	// ErrInvalidKeyBits indicates that the key size is not supported.
	ErrInvalidKeyBits = errors.New("unsupported key size")

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

func NormalizeKeyType(keyType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(keyType)) {
	case RSAKeyType, "":
		return RSAKeyType, nil
	case ECKeyType, ECDSAKeyType:
		return ECDSAKeyType, nil
	default:
		return "", ErrPrivateKeyUnsupportedType
	}
}

func ValidateKeyParams(keyType string, keyBits int) (string, int, error) {
	kt, err := NormalizeKeyType(keyType)
	if err != nil {
		return "", 0, err
	}

	switch kt {
	case RSAKeyType:
		if keyBits == 0 {
			keyBits = DefaultRSAKeyBits
		}
		if !ValidRSAKeySizes[keyBits] {
			return "", 0, fmt.Errorf("%w: RSA supports 2048 or 4096, got %d", ErrInvalidKeyBits, keyBits)
		}
	case ECDSAKeyType:
		if keyBits == 0 {
			keyBits = DefaultECDSAKeyBits
		}
		if !ValidECDSAKeySizes[keyBits] {
			return "", 0, fmt.Errorf("%w: ECDSA supports 224, 256, 384, or 521, got %d", ErrInvalidKeyBits, keyBits)
		}
	}

	return kt, keyBits, nil
}

func (a *agent) IssueCert(cn, ttl, keyType string, keyBits int) (Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	duration, err := parseTTL(ttl)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	normalizedKeyType, resolvedKeyBits, err := ValidateKeyParams(keyType, keyBits)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	var privateKey any
	var privKeyPEM string

	switch normalizedKeyType {
	case RSAKeyType:
		rsaKey, err := rsa.GenerateKey(rand.Reader, resolvedKeyBits)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = rsaKey

		privKeyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  rsaKeyBlockType,
			Bytes: privKeyBytes,
		}))

	case ECDSAKeyType:
		curve := ecdsaCurve(resolvedKeyBits)
		ecKey, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privateKey = ecKey

		privKeyBytes, err := x509.MarshalECPrivateKey(ecKey)
		if err != nil {
			return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
		}
		privKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  ecKeyBlockType,
			Bytes: privKeyBytes,
		}))
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

	if normalizedKeyType == ECDSAKeyType {
		template.KeyUsage = x509.KeyUsageDigitalSignature
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
		PrivateKeyType: normalizedKeyType,
		KeyBits:        resolvedKeyBits,
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

func ecdsaCurve(keyBits int) elliptic.Curve {
	switch keyBits {
	case 224:
		return elliptic.P224()
	case 384:
		return elliptic.P384()
	case 521:
		return elliptic.P521()
	default:
		return elliptic.P256()
	}
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
