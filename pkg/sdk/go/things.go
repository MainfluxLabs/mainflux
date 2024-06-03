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

const (
	thingsEndpoint     = "things"
	connectEndpoint    = "connect"
	disconnectEndpoint = "disconnect"
	identifyEndpoint   = "identify"
)

type identifyThingReq struct {
	Token string `json:"token,omitempty"`
}

type identifyThingResp struct {
	ID string `json:"id,omitempty"`
}

func (sdk mfSDK) CreateThing(t Thing, groupID, token string) (string, error) {
	things, err := sdk.CreateThings([]Thing{t}, groupID, token)
	if err != nil {
		return "", err
	}

	if len(things) < 1 {
		return "", nil
	}

	return things[0].ID, nil
}

func (sdk mfSDK) CreateThings(things []Thing, groupID, token string) ([]Thing, error) {
	data, err := json.Marshal(things)
	if err != nil {
		return []Thing{}, err
	}

	url := fmt.Sprintf("%s/groups/%s/%s", sdk.thingsURL, groupID, thingsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Thing{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Thing{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Thing{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Thing{}, err
	}

	var ctr createThingsRes
	if err := json.Unmarshal(body, &ctr); err != nil {
		return []Thing{}, err
	}

	return ctr.Things, nil
}

func (sdk mfSDK) Things(token string, pm PageMetadata) (ThingsPage, error) {
	url, err := sdk.withQueryParams(sdk.thingsURL, thingsEndpoint, pm)
	if err != nil {
		return ThingsPage{}, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ThingsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ThingsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ThingsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ThingsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp ThingsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ThingsPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) ThingsByChannel(token, chanID string, offset, limit uint64) (ThingsPage, error) {
	url := fmt.Sprintf("%s/channels/%s/things?offset=%d&limit=%d", sdk.thingsURL, chanID, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ThingsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ThingsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ThingsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ThingsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp ThingsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ThingsPage{}, err
	}

	return tp, nil
}

func (sdk mfSDK) Thing(id, token string) (Thing, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, id)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Thing{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Thing{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Thing{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Thing{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var t Thing
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, err
	}

	return t, nil
}

func (sdk mfSDK) UpdateThing(t Thing, thingID, token string) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, thingID)

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

func (sdk mfSDK) DeleteThing(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, id)

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

func (sdk mfSDK) DeleteThings(ids []string, token string) error {
	delReq := deleteThingsReq{ThingIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, thingsEndpoint)

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

func (sdk mfSDK) IdentifyThing(key string) (string, error) {
	idReq := identifyThingReq{Token: key}
	data, err := json.Marshal(idReq)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s", sdk.thingsURL, identifyEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	resp, err := sdk.sendRequest(req, "", string(CTJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var i identifyThingResp
	if err := json.Unmarshal(body, &i); err != nil {
		return "", err
	}

	return i.ID, err
}

func (sdk mfSDK) Connect(connIDs ConnectionIDs, token string) error {
	data, err := json.Marshal(connIDs)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, connectEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedConnect, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) Disconnect(disconnIDs ConnectionIDs, token string) error {
	data, err := json.Marshal(disconnIDs)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, disconnectEndpoint)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedDisconnect, errors.New(resp.Status))
	}

	return nil
}
