package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const webhooksEndpoint = "webhooks"

func (sdk mfSDK) CreateWebhooks(whs []Webhook, groupID, token string) ([]Webhook, error) {
	data, err := json.Marshal(whs)
	if err != nil {
		return []Webhook{}, err
	}

	url := fmt.Sprintf("%s/groups/%s/%s", sdk.webhooksURL, groupID, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return []Webhook{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return []Webhook{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return []Webhook{}, errors.Wrap(ErrFailedCreation, errors.New(resp.Status))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Webhook{}, err
	}

	var ws createWebhooksRes
	if err := json.Unmarshal(body, &ws); err != nil {
		return []Webhook{}, err
	}

	return ws.Webhooks, nil
}

func (sdk mfSDK) ListWebhooksByGroup(groupID, token string) (Webhooks, error) {
	url := fmt.Sprintf("%s/groups/%s/%s", sdk.webhooksURL, groupID, webhooksEndpoint)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Webhooks{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Webhooks{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Webhooks{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Webhooks{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var ws Webhooks
	if err := json.Unmarshal(body, &ws); err != nil {
		return Webhooks{}, err
	}

	return ws, nil
}

func (sdk mfSDK) Webhook(webhookID, token string) (Webhook, error) {
	url := fmt.Sprintf("%s/%s/%s", sdk.webhooksURL, webhooksEndpoint, webhookID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Webhook{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(CTJSON))
	if err != nil {
		return Webhook{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Webhook{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Webhook{}, errors.Wrap(ErrFailedFetch, errors.New(resp.Status))
	}

	var w Webhook
	if err := json.Unmarshal(body, &w); err != nil {
		return Webhook{}, err
	}

	return w, nil
}

func (sdk mfSDK) UpdateWebhook(wh Webhook, webhookID, token string) error {
	data, err := json.Marshal(wh)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.webhooksURL, webhooksEndpoint, webhookID)
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

func (sdk mfSDK) DeleteWebhooks(ids []string, groupID, token string) error {
	delReq := deleteWebhooksReq{WebhookIDs: ids}
	data, err := json.Marshal(delReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.webhooksURL, groupsEndpoint, groupID, webhooksEndpoint)
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
