// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const profilesEndpoint = "profiles"

func (sdk mfSDK) CreateProfile(c Profile, groupID, token string) (string, error) {
	profiles, err := sdk.CreateProfiles([]Profile{c}, groupID, token)
	if err != nil {
		return "", err
	}

	if len(profiles) < 1 {
		return "", nil
	}

	return profiles[0].ID, nil
}

func (sdk mfSDK) CreateProfiles(prs []Profile, groupID, token string) ([]Profile, error) {
	data, err := json.Marshal(prs)
	if err != nil {
		return []Profile{}, err
	}

	url := fmt.Sprintf("%s/groups/%s/%s", sdk.thingsURL, groupID, profilesEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Profile{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Profile{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Profile{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Profile{}, err
	}

	var ccr createProfilesRes
	if err := json.Unmarshal(body, &ccr); err != nil {
		return []Profile{}, err
	}

	return ccr.Profiles, nil
}

func (sdk mfSDK) Profiles(token string, pm PageMetadata) (ProfilesPage, error) {
	url, err := sdk.withQueryParams(sdk.thingsURL, profilesEndpoint, pm)
	if err != nil {
		return ProfilesPage{}, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ProfilesPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ProfilesPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ProfilesPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ProfilesPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var cp ProfilesPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ProfilesPage{}, err
	}

	return cp, nil
}

func (sdk mfSDK) ViewProfileByThing(thingID, token string) (Profile, error) {
	url := fmt.Sprintf("%s/things/%s/profiles", sdk.thingsURL, thingID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Profile{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Profile{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Profile{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Profile{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var pr Profile
	if err := json.Unmarshal(body, &pr); err != nil {
		return Profile{}, err
	}

	return pr, nil
}

func (sdk mfSDK) Profile(id, token string) (Profile, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, profilesEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Profile{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Profile{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Profile{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Profile{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var c Profile
	if err := json.Unmarshal(body, &c); err != nil {
		return Profile{}, err
	}

	return c, nil
}

func (sdk mfSDK) UpdateProfile(c Profile, profileID, token string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, profilesEndpoint, profileID)
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

func (sdk mfSDK) DeleteProfile(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, profilesEndpoint, id)
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

func (sdk mfSDK) DeleteProfiles(ids []string, token string) error {
	delReq := deleteProfilesReq{ProfileIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, profilesEndpoint)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
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
