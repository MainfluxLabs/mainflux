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

const channelsEndpoint = "channels"

func (sdk mfSDK) CreateChannel(c Channel, groupID, token string) (string, error) {
	channels, err := sdk.CreateChannels([]Channel{c}, groupID, token)
	if err != nil {
		return "", err
	}

	if len(channels) < 1 {
		return "", nil
	}

	return channels[0].ID, nil
}

func (sdk mfSDK) CreateChannels(chs []Channel, groupID, token string) ([]Channel, error) {
	data, err := json.Marshal(chs)
	if err != nil {
		return []Channel{}, err
	}

	url := fmt.Sprintf("%s/groups/%s/%s", sdk.thingsURL, groupID, channelsEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Channel{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Channel{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Channel{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Channel{}, err
	}

	var ccr createChannelsRes
	if err := json.Unmarshal(body, &ccr); err != nil {
		return []Channel{}, err
	}

	return ccr.Channels, nil
}

func (sdk mfSDK) Channels(token string, pm PageMetadata) (ChannelsPage, error) {
	url, err := sdk.withQueryParams(sdk.thingsURL, channelsEndpoint, pm)
	if err != nil {
		return ChannelsPage{}, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ChannelsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return ChannelsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ChannelsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ChannelsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var cp ChannelsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ChannelsPage{}, err
	}

	return cp, nil
}

func (sdk mfSDK) ViewChannelByThing(token, thingID string) (Channel, error) {
	url := fmt.Sprintf("%s/things/%s/channels", sdk.thingsURL, thingID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Channel{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Channel{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Channel{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Channel{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var ch Channel
	if err := json.Unmarshal(body, &ch); err != nil {
		return Channel{}, err
	}

	return ch, nil
}

func (sdk mfSDK) Channel(id, token string) (Channel, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Channel{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Channel{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Channel{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Channel{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var c Channel
	if err := json.Unmarshal(body, &c); err != nil {
		return Channel{}, err
	}

	return c, nil
}

func (sdk mfSDK) UpdateChannel(c Channel, channelID, token string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID)
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

func (sdk mfSDK) DeleteChannel(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, channelsEndpoint, id)
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

func (sdk mfSDK) DeleteChannels(ids []string, token string) error {
	delReq := deleteChannelsReq{ChannelIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, channelsEndpoint)
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
