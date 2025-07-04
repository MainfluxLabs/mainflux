// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const orgsEndpoint = "orgs"

func (sdk mfSDK) CreateOrg(o Org, token string) (string, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s", sdk.authURL, orgsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	id := strings.TrimPrefix(resp.Header.Get("Location"), fmt.Sprintf("/%s/", orgsEndpoint))
	return id, nil
}

func (sdk mfSDK) GetOrg(id, token string) (Org, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, orgsEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Org{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Org{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Org{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Org{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var o Org
	if err := json.Unmarshal(body, &o); err != nil {
		return Org{}, err
	}

	return o, nil
}

func (sdk mfSDK) UpdateOrg(o Org, orgID, token string) error {
	data, err := json.Marshal(o)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, orgsEndpoint, orgID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedUpdate, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) DeleteOrg(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.authURL, orgsEndpoint, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) ListOrgs(meta PageMetadata, token string) (OrgsPage, error) {
	u, err := url.Parse(sdk.authURL)
	if err != nil {
		return OrgsPage{}, err
	}
	u.Path = orgsEndpoint
	q := u.Query()
	q.Add("offset", strconv.FormatUint(meta.Offset, 10))
	if meta.Limit != 0 {
		q.Add("limit", strconv.FormatUint(meta.Limit, 10))
	}
	if meta.Name != "" {
		q.Add("name", meta.Name)
	}

	u.RawQuery = q.Encode()
	return sdk.getOrgs(token, u.String())
}

func (sdk mfSDK) getOrgs(token, url string) (OrgsPage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OrgsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return OrgsPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrgsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return OrgsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var op OrgsPage
	if err := json.Unmarshal(body, &op); err != nil {
		return OrgsPage{}, err
	}
	return op, nil
}

func (sdk mfSDK) ListGroupsByOrg(orgID string, meta PageMetadata, token string) (GroupsPage, error) {
	apiUrl := fmt.Sprintf("%s/%s/%s/groups?offset=%d&limit=%d", sdk.thingsURL, orgsEndpoint, orgID, meta.Offset, meta.Limit)

	req, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		return GroupsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupsPage{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GroupsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var gp GroupsPage
	if err := json.Unmarshal(body, &gp); err != nil {
		return GroupsPage{}, err
	}

	return gp, nil
}
