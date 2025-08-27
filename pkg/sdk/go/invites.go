package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const redirectPath = "/view-invite"

const (
	UserTypeInviter = "inviter"
	UserTypeInvitee = "invitee"
)

func (sdk mfSDK) CreateInvite(orgID string, om OrgMembership, token string) (Invite, error) {
	data, err := json.Marshal(struct {
		Om           OrgMembership `json:"org_membership"`
		RedirectPath string        `json:"redirect_path"`
	}{
		om,
		redirectPath,
	})

	if err != nil {
		return Invite{}, err
	}

	url := fmt.Sprintf("%s/orgs/%s/invites", sdk.authURL, orgID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return Invite{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Invite{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Invite{}, err
	}

	if resp.StatusCode != http.StatusCreated {
		return Invite{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	var inv Invite
	if err := json.Unmarshal(body, &inv); err != nil {
		return Invite{}, err
	}

	return inv, nil
}

func (sdk mfSDK) RevokeInvite(inviteID string, token string) error {
	url := fmt.Sprintf("%s/invites/%s", sdk.authURL, inviteID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, "")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.Wrap(ErrFailedRemoval, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) InviteRespond(inviteID string, accept bool, token string) error {
	var responseVerb string
	if accept {
		responseVerb = "accept"
	} else {
		responseVerb = "decline"
	}

	url := fmt.Sprintf("%s/invites/%s/%s", sdk.authURL, inviteID, responseVerb)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, "")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	return nil
}

func (sdk mfSDK) GetInvite(inviteID string, token string) (Invite, error) {
	url := fmt.Sprintf("%s/invites/%s", sdk.authURL, inviteID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Invite{}, err
	}

	resp, err := sdk.sendRequest(req, token, "")
	if err != nil {
		return Invite{}, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Invite{}, err
	}

	var inv Invite

	if err := json.Unmarshal(body, &inv); err != nil {
		return Invite{}, err
	}

	return inv, nil
}

func (sdk mfSDK) ListInvitesByUser(userID string, userType string, pm PageMetadata, token string) (InvitesPage, error) {
	url := fmt.Sprintf("%s/users/%s/invites", sdk.authURL, userID)
	switch userType {
	case UserTypeInviter:
		url += "/sent"
	case UserTypeInvitee:
		url += "/received"
	default:
		url += "/sent"
	}

	url, err := sdk.withQueryParams(url, "", pm)
	if err != nil {
		return InvitesPage{}, err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return InvitesPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, "")
	if err != nil {
		return InvitesPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return InvitesPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return InvitesPage{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}
	var ip InvitesPage
	if err := json.Unmarshal(body, &ip); err != nil {
		return InvitesPage{}, err
	}

	return ip, nil
}
