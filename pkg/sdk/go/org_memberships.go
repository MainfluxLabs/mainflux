package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const membershipsEndpoint = "memberships"

func (sdk mfSDK) CreateOrgMemberships(om []OrgMembership, orgID string, token string) error {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.authURL, orgsEndpoint, orgID, membershipsEndpoint)

	omr := orgMembershipsReq{
		OrgMembers: om,
	}

	data, err := json.Marshal(omr)
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

func (sdk mfSDK) GetOrgMembership(memberID, orgID, token string) (OrgMembership, error) {
	url := fmt.Sprintf("%s/%s/%s/members/%s", sdk.authURL, orgsEndpoint, orgID, memberID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OrgMembership{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return OrgMembership{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrgMembership{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return OrgMembership{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var membership OrgMembership
	if err := json.Unmarshal(body, &membership); err != nil {
		return OrgMembership{}, err
	}

	return membership, nil
}

func (sdk mfSDK) ListOrgMemberships(orgID string, pm PageMetadata, token string) (OrgMembershipsPage, error) {
	url := fmt.Sprintf("%s/%s/%s/%s?offset=%d&limit=%d", sdk.authURL, orgsEndpoint, orgID, membershipsEndpoint, pm.Offset, pm.Limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OrgMembershipsPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return OrgMembershipsPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return OrgMembershipsPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return OrgMembershipsPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var mp OrgMembershipsPage
	if err := json.Unmarshal(body, &mp); err != nil {
		return OrgMembershipsPage{}, err
	}

	return mp, nil
}

func (sdk mfSDK) UpdateOrgMemberships(oms []OrgMembership, orgID, token string) error {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.authURL, orgsEndpoint, orgID, membershipsEndpoint)
	omr := orgMembershipsReq{
		OrgMembers: oms,
	}

	data, err := json.Marshal(omr)
	if err != nil {
		return err
	}

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

func (sdk mfSDK) RemoveOrgMemberships(memberIDs []string, orgID, token string) error {
	var ids []string
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.authURL, orgsEndpoint, orgID, membershipsEndpoint)
	ids = append(ids, memberIDs...)
	rmr := removeMembershipReq{
		MemberIDs: ids,
	}

	data, err := json.Marshal(rmr)
	if err != nil {
		return err
	}

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
