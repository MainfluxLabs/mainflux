package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

func (sdk mfSDK) CreateGroupMemberships(gms []GroupMembership, groupID, token string) error {
	data, err := json.Marshal(gms)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membershipsEndpoint)

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

func (sdk mfSDK) UpdateGroupMemberships(gms []GroupMembership, groupID, token string) error {
	data, err := json.Marshal(gms)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membershipsEndpoint)
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

func (sdk mfSDK) RemoveGroupMemberships(ids []string, groupID, token string) error {
	delReq := removeMembershipsReq{MemberIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, groupsEndpoint, groupID, membershipsEndpoint)

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

func (sdk mfSDK) ListGroupMemberships(groupID string, pm PageMetadata, token string) (GroupMembershipsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/%s?offset=%d&limit=%d", sdk.thingsURL, groupsEndpoint, groupID, membershipsEndpoint, pm.Offset, pm.Limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return GroupMembershipsPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return GroupMembershipsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var gmp GroupMembershipsPage
	if err := json.Unmarshal(body, &gmp); err != nil {
		return GroupMembershipsPage{}, err
	}

	return gmp, nil
}
