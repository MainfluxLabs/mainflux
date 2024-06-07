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

const groupsEndpoint = "groups"

func (sdk mfSDK) CreateGroup(g Group, orgID, token string) (string, error) {
	groups, err := sdk.CreateGroups([]Group{g}, orgID, token)
	if err != nil {
		return "", err
	}

	if len(groups) < 1 {
		return "", nil
	}

	return groups[0].ID, nil
}

func (sdk mfSDK) CreateGroups(groups []Group, orgID, token string) ([]Group, error) {
	data, err := json.Marshal(groups)
	if err != nil {
		return []Group{}, err
	}

	url := fmt.Sprintf("%s/orgs/%s/%s", sdk.thingsURL, orgID, groupsEndpoint)

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

func (sdk mfSDK) ListThingsByGroup(groupID, token string, offset, limit uint64) (ThingsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/things?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, offset, limit)
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

	var gtp ThingsPage
	if err := json.Unmarshal(body, &gtp); err != nil {
		return ThingsPage{}, err
	}

	return gtp, nil
}

func (sdk mfSDK) ListChannelsByGroup(groupID, token string, offset, limit uint64) (ChannelsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/channels?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, offset, limit)
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

	var gcp ChannelsPage
	if err := json.Unmarshal(body, &gcp); err != nil {
		return ChannelsPage{}, err
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

func (sdk mfSDK) UpdateGroup(g Group, groupID, token string) error {
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID)
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

func (sdk mfSDK) ViewGroupByThing(thingID, token string) (Group, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, thingsEndpoint, thingID, groupsEndpoint)
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

func (sdk mfSDK) ViewGroupByChannel(channelID, token string) (Group, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, channelsEndpoint, channelID, groupsEndpoint)
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

func (sdk mfSDK) CreateRolesByGroup(roles []GroupMember, groupID, token string) error {
	data, err := json.Marshal(roles)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membersEndpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) UpdateRolesByGroup(roles []GroupMember, groupID, token string) error {
	data, err := json.Marshal(roles)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membersEndpoint)
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

func (sdk mfSDK) RemoveRolesByGroup(ids []string, groupID, token string) error {
	delReq := unassignMemberReq{MemberIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membersEndpoint)

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

func (sdk mfSDK) ListRolesByGroup(groupID, token string, offset, limit uint64) (GroupRolesPage, error) {
	url := fmt.Sprintf("%s/%s/%s/members?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, offset, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupRolesPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupRolesPage{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GroupRolesPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupRolesPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var grp GroupRolesPage
	if err := json.Unmarshal(body, &grp); err != nil {
		return GroupRolesPage{}, err
	}

	return grp, nil
}
