// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	groupsEndpoint = "groups"
	MaxLevel       = uint64(5)
	MinLevel       = uint64(1)
)

func (sdk mfSDK) CreateGroup(g Group, token string) (string, error) {
	groups, err := sdk.CreateGroups([]Group{g}, token)
	if err != nil {
		return "", err
	}

	if len(groups) < 1 {
		return "", nil
	}

	return groups[0].ID, nil
}

func (sdk mfSDK) CreateGroups(groups []Group, token string) ([]Group, error) {
	data, err := json.Marshal(groups)
	if err != nil {
		return []Group{}, err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, groupsEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Group{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Group{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Group{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Group{}, err
	}

	var cgr createGroupsRes
	if err := json.Unmarshal(body, &cgr); err != nil {
		return []Group{}, err
	}

	return cgr.Groups, nil
}

func (sdk mfSDK) DeleteGroup(id, token string) error {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, groupsEndpoint, id)
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

func (sdk mfSDK) DeleteGroups(ids []string, token string) error {
	delReq := deleteGroupsReq{GroupIDs: ids}

	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, groupsEndpoint)
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

func (sdk mfSDK) AssignThing(memberIDs []string, groupID string, token string) error {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/things", sdk.thingsURL, groupsEndpoint, groupID)
	ids = append(ids, memberIDs...)
	assignThingReq := groupThingsReq{
		Things: ids,
	}

	data, err := json.Marshal(assignThingReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrMemberAdd, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) UnassignThing(token, groupID string, thingIDs ...string) error {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/things", sdk.thingsURL, groupsEndpoint, groupID)
	ids = append(ids, thingIDs...)
	unassignThingReq := groupThingsReq{
		Things: ids,
	}

	data, err := json.Marshal(unassignThingReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(data))
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

func (sdk mfSDK) ListGroupThings(groupID, token string, offset, limit uint64) (GroupThingsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/things?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupThingsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupThingsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupThingsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupThingsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var gtp GroupThingsPage
	if err := json.Unmarshal(body, &gtp); err != nil {
		return GroupThingsPage{}, err
	}

	return gtp, nil
}

func (sdk mfSDK) AssignChannel(channelIDs []string, groupID string, token string) error {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/channels", sdk.thingsURL, groupsEndpoint, groupID)
	ids = append(ids, channelIDs...)
	assignChannelReq := groupChannelsReq{
		Channels: ids,
	}

	data, err := json.Marshal(assignChannelReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrMemberAdd, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) UnassignChannel(token, groupID string, thingIDs ...string) error {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/channels", sdk.thingsURL, groupsEndpoint, groupID)
	ids = append(ids, thingIDs...)
	unassignChannelReq := groupChannelsReq{
		Channels: ids,
	}

	data, err := json.Marshal(unassignChannelReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader(data))
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

func (sdk mfSDK) ListGroupChannels(groupID, token string, offset, limit uint64) (GroupChannelsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/channels?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupChannelsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupChannelsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupChannelsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupChannelsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var gcp GroupChannelsPage
	if err := json.Unmarshal(body, &gcp); err != nil {
		return GroupChannelsPage{}, err
	}

	return gcp, nil
}

func (sdk mfSDK) Groups(meta PageMetadata, token string) (GroupsPage, error) {
	u, err := url.Parse(sdk.thingsURL)
	if err != nil {
		return GroupsPage{}, err
	}
	u.Path = groupsEndpoint
	q := u.Query()
	q.Add("offset", strconv.FormatUint(meta.Offset, 10))
	if meta.Limit != 0 {
		q.Add("limit", strconv.FormatUint(meta.Limit, 10))
	}
	if meta.Level != 0 {
		q.Add("level", strconv.FormatUint(meta.Level, 10))
	}
	if meta.Name != "" {
		q.Add("name", meta.Name)
	}
	if meta.Type != "" {
		q.Add("type", meta.Type)
	}
	u.RawQuery = q.Encode()
	return sdk.getGroups(token, u.String())
}

func (sdk mfSDK) getGroups(token, url string) (GroupsPage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupsPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var tp GroupsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return GroupsPage{}, err
	}
	return tp, nil
}

func (sdk mfSDK) Group(id, token string) (Group, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, groupsEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Group{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Group{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Group{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Group{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var t Group
	if err := json.Unmarshal(body, &t); err != nil {
		return Group{}, err
	}

	return t, nil
}

func (sdk mfSDK) UpdateGroup(t Group, token string) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, groupsEndpoint, t.ID)
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

func (sdk mfSDK) ViewThingMembership(thingID, token string, offset, limit uint64) (Group, error) {
	url := fmt.Sprintf("%s/%s/%s/%s?offset=%d&limit=%d", sdk.thingsURL, thingsEndpoint, thingID, groupsEndpoint, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Group{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Group{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Group{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Group{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var g Group
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, err
	}

	return g, nil
}

func (sdk mfSDK) ViewChannelMembership(channelID, token string, offset, limit uint64) (Group, error) {
	url := fmt.Sprintf("%s/%s/%s/%s?offset=%d&limit=%d", sdk.thingsURL, channelsEndpoint, channelID, groupsEndpoint, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Group{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Group{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Group{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Group{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var g Group
	if err := json.Unmarshal(body, &g); err != nil {
		return Group{}, err
	}

	return g, nil
}
