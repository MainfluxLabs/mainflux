package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const certsEndpoint = "certs"

// Cert represents certs data.
type Cert struct {
	ThingID        string    `json:"thing_id,omitempty"`
	Certificate    string    `json:"certificate,omitempty"`
	PrivateKey     string    `json:"private_key,omitempty"`
	IssuingCA      string    `json:"issuing_ca,omitempty"`
	CAChain        []string  `json:"ca_chain,omitempty"`
	PrivateKeyType string    `json:"private_key_type,omitempty"`
	Serial         string    `json:"serial,omitempty"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
}

// CertSerial represents a certificate serial entry in a listing.
type CertSerial struct {
	ThingID    string    `json:"thing_id"`
	CertSerial string    `json:"cert_serial"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// CertsPage contains a page of certificate serials.
type CertsPage struct {
	Total  uint64       `json:"total"`
	Offset uint64       `json:"offset"`
	Limit  uint64       `json:"limit"`
	Certs  []CertSerial `json:"certs"`
}

func (sdk mfSDK) IssueCert(thingID string, keyBits int, keyType, valid, token string) (Cert, error) {
	var c Cert
	r := certReq{
		ThingID: thingID,
		KeyBits: keyBits,
		KeyType: keyType,
		TTL:     valid,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.certsURL, certsEndpoint)
	res, err := request(http.MethodPost, token, url, d)
	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return Cert{}, ErrCerts
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Cert{}, err
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}
	return c, nil
}

func (sdk mfSDK) ViewCert(serial, token string) (Cert, error) {
	var c Cert
	url := fmt.Sprintf("%s/%s/%s", sdk.certsURL, certsEndpoint, serial)
	res, err := request(http.MethodGet, token, url, nil)
	if err != nil {
		return Cert{}, err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Cert{}, ErrCerts
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Cert{}, err
	}

	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}
	return c, nil
}

func (sdk mfSDK) RevokeCert(serial, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.certsURL, certsEndpoint, serial)
	res, err := request(http.MethodDelete, token, url, nil)

	if res != nil {
		res.Body.Close()
	}

	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusForbidden:
		return errors.ErrAuthorization
	default:
		return ErrCertsRemove
	}
}

func (sdk mfSDK) RenewCert(serial, token string) (Cert, error) {
	var c Cert
	url := fmt.Sprintf("%s/%s/%s", sdk.certsURL, certsEndpoint, serial)
	res, err := request(http.MethodPut, token, url, nil)

	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return Cert{}, ErrCerts
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Cert{}, err
	}

	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}

	return c, nil
}

func (sdk mfSDK) ListSerials(thingID string, offset, limit uint64, token string) (CertsPage, error) {
	var cp CertsPage
	url := fmt.Sprintf("%s/svccerts/things/%s/serials?offset=%d&limit=%d", sdk.certsURL, thingID, offset, limit)
	res, err := request(http.MethodGet, token, url, nil)
	if err != nil {
		return CertsPage{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return CertsPage{}, err
	}

	if res.StatusCode != http.StatusOK {
		return CertsPage{}, fmt.Errorf("status %d: %s", res.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &cp); err != nil {
		return CertsPage{}, err
	}

	return cp, nil
}

func (sdk mfSDK) RemoveCert(id, token string) error {
	res, err := request(http.MethodDelete, token, fmt.Sprintf("%s/%s", sdk.certsURL, id), nil)
	if res != nil {
		res.Body.Close()
	}
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusForbidden:
		return errors.ErrAuthorization
	default:
		return ErrCertsRemove
	}
}

func request(method, jwt, url string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	c := &http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type certReq struct {
	ThingID string `json:"thing_id"`
	KeyBits int    `json:"key_bits"`
	KeyType string `json:"key_type"`
	TTL     string `json:"ttl"`
}
